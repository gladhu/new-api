package middleware

import (
	"net/http/httptest"
	"testing"

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
}
