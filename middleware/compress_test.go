package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNegotiateEncoding(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "br", negotiateEncoding("br"))
	assert.Equal(t, "br", negotiateEncoding("br, gzip;q=0.8"))
	assert.Equal(t, "gzip", negotiateEncoding("gzip"))
	assert.Equal(t, "gzip", negotiateEncoding("deflate, gzip, br;q=0.1"))
	assert.Equal(t, "", negotiateEncoding(""))
	assert.Equal(t, "", negotiateEncoding("deflate"))
}

func TestShouldSkipContentType(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldSkipContentType("image/png"))
	assert.True(t, shouldSkipContentType("font/woff2"))
	assert.True(t, shouldSkipContentType("application/font-woff2"))
	assert.False(t, shouldSkipContentType("text/javascript; charset=utf-8"))
	assert.False(t, shouldSkipContentType("application/json"))
}

func TestCompressPrefersBrotli(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Compress())
	router.GET("/hello", func(c *gin.Context) {
		c.Data(200, "text/plain; charset=utf-8", []byte(string(make([]byte, compressMinLength))))
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
	assert.Equal(t, "br", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))
	assert.NotEmpty(t, rec.Body.Bytes())
}

func TestCompressFallsBackToGzip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Compress())
	router.GET("/hello", func(c *gin.Context) {
		c.Data(200, "text/plain; charset=utf-8", []byte(string(make([]byte, compressMinLength))))
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
}

func TestCompressSkipsSmallResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Compress())
	router.GET("/small", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/small", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "ok", rec.Body.String())
}

func TestCompressSkipsAlreadyCompressedContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Compress())
	router.GET("/logo.png", func(c *gin.Context) {
		c.Data(200, "image/png", []byte(string(make([]byte, compressMinLength))))
	})

	req := httptest.NewRequest("GET", "/logo.png", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
}

func TestCompressPreservesLargeJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	want := strings.Repeat(`{"id":1,"name":"user","email":"test@example.com"},`, 200)
	router := gin.New()
	router.Use(Compress())
	router.GET("/users", func(c *gin.Context) {
		c.Data(200, "application/json; charset=utf-8", []byte("["+want+"]"))
	})

	for _, encoding := range []string{"br", "gzip"} {
		t.Run(encoding, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users", nil)
			req.Header.Set("Accept-Encoding", encoding)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, 200, rec.Code)
			assert.Equal(t, encoding, rec.Header().Get("Content-Encoding"))

			body, err := decompressBody(encoding, rec.Body.Bytes())
			require.NoError(t, err)
			assert.Equal(t, "["+want+"]", string(body))
		})
	}
}

// Content-Type is set by gin renderers after c.Status, so the middleware must not
// freeze or rebuild the header map when the status code is recorded.
func TestCompressPreservesContentTypeAndHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Compress())
	router.GET("/large", func(c *gin.Context) {
		c.Header("X-Trace-Id", "trace-1")
		c.JSON(http.StatusOK, gin.H{"data": strings.Repeat("y", compressMinLength)})
	})
	router.GET("/small", func(c *gin.Context) {
		c.Header("X-Trace-Id", "trace-2")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	cases := []struct {
		path            string
		wantEncoding    string
		wantTraceID     string
		wantContentType string
	}{
		{path: "/large", wantEncoding: "gzip", wantTraceID: "trace-1", wantContentType: "application/json; charset=utf-8"},
		{path: "/small", wantEncoding: "", wantTraceID: "trace-2", wantContentType: "application/json; charset=utf-8"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tc.wantEncoding, rec.Header().Get("Content-Encoding"))
			assert.Equal(t, tc.wantContentType, rec.Header().Get("Content-Type"))
			assert.Equal(t, tc.wantTraceID, rec.Header().Get("X-Trace-Id"))
		})
	}
}

// gin renders redirects as Status(-1) followed by http.Redirect writing the real
// status and Location header, so the middleware must not latch the first status.
func TestCompressPreservesRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Compress())
	router.GET("/pay/return", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/wallet?pay=success")
	})

	req := httptest.NewRequest(http.MethodGet, "/pay/return", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/wallet?pay=success", rec.Header().Get("Location"))
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
}

// Streaming endpoints (e.g. Ollama pull progress) emit events far below
// compressMinLength and must not stay stuck in the compression buffer.
func TestCompressFlushesStreamedEventBeforeHandlerReturns(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	seenAfterFlush := ""

	router := gin.New()
	router.Use(Compress())
	router.GET("/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(c.Writer, "data: progress\n\n")
		c.Writer.Flush()
		seenAfterFlush = rec.Body.String()
		_, _ = fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	})

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "data: progress\n\n", seenAfterFlush)
	assert.Equal(t, "data: progress\n\ndata: [DONE]\n\n", rec.Body.String())
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
}

func TestCompressSkipsRangeRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(strings.Repeat("static-javascript;", 200))
	router := gin.New()
	router.Use(Compress())
	router.GET("/static/js/async/chunk.js", func(c *gin.Context) {
		http.ServeContent(
			c.Writer,
			c.Request,
			"chunk.js",
			time.Time{},
			bytes.NewReader(body),
		)
	})

	req := httptest.NewRequest(http.MethodGet, "/static/js/async/chunk.js", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	req.Header.Set("Range", "bytes=0-1023")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusPartialContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, body[:1024], rec.Body.Bytes())
}

func TestCompressesServeContentStaticAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(strings.Repeat("static-javascript;", 200))
	router := gin.New()
	router.Use(Compress())
	router.GET("/static/js/async/chunk.js", func(c *gin.Context) {
		http.ServeContent(
			c.Writer,
			c.Request,
			"chunk.js",
			time.Time{},
			bytes.NewReader(body),
		)
	})

	req := httptest.NewRequest(http.MethodGet, "/static/js/async/chunk.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Empty(t, rec.Header().Get("Content-Length"))

	decompressed, err := decompressBody("gzip", rec.Body.Bytes())
	require.NoError(t, err)
	assert.Equal(t, body, decompressed)
}

func decompressBody(encoding string, data []byte) ([]byte, error) {
	switch encoding {
	case "br":
		return io.ReadAll(brotli.NewReader(bytes.NewReader(data)))
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	default:
		return data, nil
	}
}
