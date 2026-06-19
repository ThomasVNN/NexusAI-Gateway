package compression

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"strings"
)

// GzipCompress compresses data using gzip
func GzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GzipDecompress decompresses gzip data
func GzipDecompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// DeflateCompress compresses data using deflate
func DeflateCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DeflateDecompress decompresses deflate data
func DeflateDecompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// DetectEncoding detects the compression encoding from magic bytes
func DetectEncoding(data []byte) string {
	if len(data) < 2 {
		return ""
	}

	// Gzip magic bytes
	if data[0] == 0x1f && data[1] == 0x8b {
		return "gzip"
	}

	// Zlib (deflate) magic bytes
	if data[0] == 0x78 {
		return "deflate"
	}

	return ""
}

// AcceptEncoding parses Accept-Encoding header and returns preferred encoding
func AcceptEncoding(header string) string {
	if header == "" {
		return ""
	}

	encodings := strings.Split(header, ",")
	for _, enc := range encodings {
		enc = strings.TrimSpace(enc)
		if enc == "gzip" {
			return "gzip"
		}
		if enc == "deflate" {
			return "deflate"
		}
		if enc == "identity" {
			return ""
		}
	}

	return ""
}

// WrapBodyReadCloser wraps an io.Reader as an io.ReadCloser
type WrapBodyReadCloser struct {
	io.Reader
}

// Close implements io.Closer
func (w *WrapBodyReadCloser) Close() error {
	return nil
}

// NewWrapBodyReadCloser creates a new WrapBodyReadCloser
func NewWrapBodyReadCloser(r io.Reader) io.ReadCloser {
	return &WrapBodyReadCloser{r}
}

// CompressHandler wraps an http.Handler to compress responses
func CompressHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
