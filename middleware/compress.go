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
	if c.GetHeader("Range") != "" {
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
	encoding  string
	minLength int
	buf       bytes.Buffer
	writer    io.WriteCloser
	skipped   bool
}

func newCompressResponseWriter(w gin.ResponseWriter, encoding string) *compressResponseWriter {
	return &compressResponseWriter{
		ResponseWriter: w,
		encoding:       encoding,
		minLength:      compressMinLength,
	}
}

// WriteHeaderNow commits the response header, so buffered bytes can no longer be
// labelled with Content-Encoding and must be sent uncompressed.
func (w *compressResponseWriter) WriteHeaderNow() {
	if w.writer == nil {
		w.skipped = true
		_ = w.writeRawBuffer()
	}
	w.ResponseWriter.WriteHeaderNow()
}

func (w *compressResponseWriter) Write(data []byte) (int, error) {
	n := len(data)

	if w.writer != nil {
		return w.writer.Write(data)
	}
	if w.skipped {
		return w.ResponseWriter.Write(data)
	}

	w.buf.Write(data)
	if w.buf.Len() < w.minLength {
		return n, nil
	}

	if err := w.startCompression(); err != nil {
		return 0, err
	}
	// startCompression already flushed buf (including this write); do not write data again.
	return n, nil
}

func (w *compressResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *compressResponseWriter) Flush() {
	if w.writer != nil {
		if flusher, ok := w.writer.(interface{ Flush() error }); ok {
			_ = flusher.Flush()
		}
	} else if !w.skipped && w.buf.Len() > 0 {
		if w.buf.Len() >= w.minLength {
			_ = w.startCompression()
		} else {
			// A flush means the client expects the bytes now, so stop buffering
			// for the rest of the response instead of waiting for minLength.
			w.skipped = true
			_ = w.writeRawBuffer()
		}
	}

	w.ResponseWriter.Flush()
}

func (w *compressResponseWriter) startCompression() error {
	if w.writer != nil {
		return nil
	}

	if !w.shouldCompressResponse() {
		w.skipped = true
		return w.writeRawBuffer()
	}

	var compressor io.WriteCloser
	switch w.encoding {
	case "br":
		compressor = brotli.NewWriterLevel(w.ResponseWriter, brotliQuality)
	case "gzip":
		gz, err := gzip.NewWriterLevel(w.ResponseWriter, gzip.DefaultCompression)
		if err != nil {
			return err
		}
		compressor = gz
	default:
		w.skipped = true
		return w.writeRawBuffer()
	}

	// Safe until the first byte reaches the client: gin buffers the status line
	// and only emits headers on the first underlying write.
	header := w.ResponseWriter.Header()
	header.Del("Content-Length")
	header.Set("Content-Encoding", w.encoding)
	w.writer = compressor

	if w.buf.Len() > 0 {
		if _, err := w.writer.Write(w.buf.Bytes()); err != nil {
			return err
		}
		w.buf.Reset()
	}
	return nil
}

func (w *compressResponseWriter) shouldCompressResponse() bool {
	status := w.ResponseWriter.Status()
	if status < http.StatusOK || status == http.StatusNoContent ||
		status == http.StatusNotModified || status == http.StatusPartialContent {
		return false
	}

	header := w.Header()
	if shouldSkipContentType(header.Get("Content-Type")) {
		return false
	}
	if header.Get("Content-Encoding") != "" || header.Get("Content-Range") != "" {
		return false
	}
	return !strings.Contains(strings.ToLower(header.Get("Cache-Control")), "no-transform")
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

	if !w.skipped && w.buf.Len() >= w.minLength {
		if err := w.startCompression(); err == nil && w.writer != nil {
			_ = w.writer.Close()
			return
		}
	}

	w.skipped = true
	_ = w.writeRawBuffer()
}
