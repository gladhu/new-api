package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

const (
	compressMinLength = 1024
	brotliQuality     = 5
)

// Compress negotiates Brotli or gzip for HTML/API/static responses on web routes.
// Relay/streaming routers must not use this middleware.
func Compress() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !shouldCompressRequest(c) {
			c.Next()
			return
		}

		encoding := negotiateEncoding(c.GetHeader("Accept-Encoding"))
		if encoding == "" {
			c.Next()
			return
		}

		cw := newCompressResponseWriter(c.Writer, encoding)
		c.Writer = cw
		c.Header("Vary", "Accept-Encoding")
		c.Next()
		cw.finish()
	}
}

func shouldCompressRequest(c *gin.Context) bool {
	if c.Request.Method == http.MethodHead {
		return false
	}
	if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		return false
	}
	if c.GetHeader("Accept-Encoding") == "" {
		return false
	}
	return true
}

func negotiateEncoding(acceptEncoding string) string {
	if acceptEncoding == "" {
		return ""
	}

	brQ := -1.0
	gzipQ := -1.0
	for _, part := range strings.Split(acceptEncoding, ",") {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}

		encoding := token
		quality := 1.0
		if before, after, ok := strings.Cut(token, ";"); ok {
			encoding = strings.TrimSpace(before)
			after = strings.TrimSpace(after)
			if qValue, ok := strings.CutPrefix(strings.ToLower(after), "q="); ok {
				if parsed, err := strconv.ParseFloat(strings.TrimSpace(qValue), 64); err == nil {
					quality = parsed
				}
			}
		}

		switch strings.ToLower(encoding) {
		case "br":
			if quality > brQ {
				brQ = quality
			}
		case "gzip":
			if quality > gzipQ {
				gzipQ = quality
			}
		}
	}

	if brQ <= 0 && gzipQ <= 0 {
		return ""
	}
	if brQ >= gzipQ && brQ > 0 {
		return "br"
	}
	if gzipQ > 0 {
		return "gzip"
	}
	return ""
}

func shouldSkipContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "" {
		return false
	}
	if before, _, ok := strings.Cut(contentType, ";"); ok {
		contentType = strings.TrimSpace(before)
	}

	skipPrefixes := []string{
		"image/",
		"video/",
		"audio/",
		"font/",
		"application/zip",
		"application/gzip",
		"application/x-gzip",
		"application/x-compress",
		"application/x-bzip2",
		"application/x-xz",
		"application/zstd",
		"application/pdf",
	}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}
	if strings.Contains(contentType, "font-woff") || strings.Contains(contentType, "font/woff") {
		return true
	}
	return false
}

type compressResponseWriter struct {
	gin.ResponseWriter
	encoding      string
	minLength     int
	buf           bytes.Buffer
	writer        io.WriteCloser
	wroteHeader   bool
	headerWritten bool
	skipped       bool
}

func newCompressResponseWriter(w gin.ResponseWriter, encoding string) *compressResponseWriter {
	return &compressResponseWriter{
		ResponseWriter: w,
		encoding:         encoding,
		minLength:        compressMinLength,
	}
}

func (w *compressResponseWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}
	w.wroteHeader = true
	w.headerWritten = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *compressResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.writer != nil {
		return w.writer.Write(data)
	}
	if w.skipped {
		return w.ResponseWriter.Write(data)
	}

	w.buf.Write(data)
	if w.buf.Len() < w.minLength {
		return len(data), nil
	}

	if err := w.startCompression(); err != nil {
		return 0, err
	}
	if w.writer != nil {
		return w.writer.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *compressResponseWriter) startCompression() error {
	if w.writer != nil {
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if shouldSkipContentType(contentType) {
		w.skipped = true
		return w.writeRawBuffer()
	}

	header := w.ResponseWriter.Header()
	header.Del("Content-Length")
	header.Set("Content-Encoding", w.encoding)

	var underlying io.Writer = w.ResponseWriter
	switch w.encoding {
	case "br":
		w.writer = brotli.NewWriterLevel(underlying, brotliQuality)
	case "gzip":
		gz, err := gzip.NewWriterLevel(underlying, gzip.DefaultCompression)
		if err != nil {
			return err
		}
		w.writer = gz
	default:
		w.skipped = true
		return w.writeRawBuffer()
	}

	if w.buf.Len() > 0 {
		if _, err := w.writer.Write(w.buf.Bytes()); err != nil {
			return err
		}
		w.buf.Reset()
	}
	return nil
}

func (w *compressResponseWriter) writeRawBuffer() error {
	if w.buf.Len() == 0 {
		return nil
	}
	_, err := w.ResponseWriter.Write(w.buf.Bytes())
	w.buf.Reset()
	return err
}

func (w *compressResponseWriter) finish() {
	if w.writer != nil {
		_ = w.writer.Close()
		return
	}

	if w.buf.Len() == 0 {
		return
	}

	if w.skipped {
		_ = w.writeRawBuffer()
		return
	}

	if w.buf.Len() >= w.minLength && !shouldSkipContentType(w.Header().Get("Content-Type")) {
		if err := w.startCompression(); err == nil && w.writer != nil {
			_, _ = w.writer.Write(w.buf.Bytes())
			w.buf.Reset()
			_ = w.writer.Close()
			return
		}
	}

	_ = w.writeRawBuffer()
}
