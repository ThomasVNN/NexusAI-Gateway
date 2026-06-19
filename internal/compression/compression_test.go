package compression

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockCompressor is a simple test compressor
type MockCompressor struct {
	data []byte
}

func (c *MockCompressor) Compress(data []byte) ([]byte, error) {
	return bytes.Repeat([]byte{0x00}, len(data)/2), nil
}

func (c *MockCompressor) Decompress(data []byte) ([]byte, error) {
	return data, nil
}

func TestCompressMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Test without compression
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestGzipCompress(t *testing.T) {
	data := []byte("test data for compression")

	compressed, err := GzipCompress(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	decompressed, err := GzipDecompress(compressed)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if string(decompressed) != string(data) {
		t.Errorf("Expected '%s', got '%s'", string(data), string(decompressed))
	}
}

func TestGzipCompressLargeData(t *testing.T) {
	// Create 1MB of data
	data := bytes.Repeat([]byte("test data for compression "), 10000)

	compressed, err := GzipCompress(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	decompressed, err := GzipDecompress(compressed)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(decompressed) != len(data) {
		t.Errorf("Expected length %d, got %d", len(data), len(decompressed))
	}
}

func TestGzipDecompressInvalid(t *testing.T) {
	_, err := GzipDecompress([]byte("invalid gzip data"))
	if err == nil {
		t.Error("Expected error for invalid gzip data")
	}
}

func TestDeflateCompress(t *testing.T) {
	data := []byte("test data for compression")

	compressed, err := DeflateCompress(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	decompressed, err := DeflateDecompress(compressed)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if string(decompressed) != string(data) {
		t.Errorf("Expected '%s', got '%s'", string(data), string(decompressed))
	}
}

func TestDetectEncoding(t *testing.T) {
	// Test gzip
	gzipData := []byte{0x1f, 0x8b, 0x08, 0x00}
	if enc := DetectEncoding(gzipData); enc != "gzip" {
		t.Errorf("Expected 'gzip', got '%s'", enc)
	}

	// Test deflate
	deflateData := []byte{0x78, 0x9c}
	if enc := DetectEncoding(deflateData); enc != "deflate" {
		t.Errorf("Expected 'deflate', got '%s'", enc)
	}

	// Test unknown
	unknownData := []byte("plain text")
	if enc := DetectEncoding(unknownData); enc != "" {
		t.Errorf("Expected '', got '%s'", enc)
	}
}

func TestAcceptEncoding(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"gzip, deflate", "gzip"},
		{"deflate, gzip", "deflate"},
		{"gzip", "gzip"},
		{"deflate", "deflate"},
		{"", ""},
		{"identity", ""},
	}

	for _, tt := range tests {
		got := AcceptEncoding(tt.header)
		if got != tt.want {
			t.Errorf("AcceptEncoding(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestWrapBodyReadCloser(t *testing.T) {
	original := bytes.NewReader([]byte("test data"))
	wrapper := NewWrapBodyReadCloser(original)

	// Read from wrapper
	data := make([]byte, 100)
	n, err := wrapper.Read(data)
	if err != nil && err != io.EOF {
		t.Errorf("Unexpected error: %v", err)
	}
	if n == 0 {
		t.Error("Expected to read some data")
	}

	// Close should succeed
	err = wrapper.Close()
	if err != nil {
		t.Errorf("Unexpected close error: %v", err)
	}
}

func TestCompressHandler(t *testing.T) {
	// Create a handler that returns a response
	handler := CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}
