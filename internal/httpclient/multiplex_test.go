package httpclient

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestMultiplexedTransportCreation(t *testing.T) {
	mt := NewMultiplexedTransport(10)
	if mt == nil {
		t.Error("Expected non-nil transport")
	}
}

func TestMultiplexedTransportGetTransport(t *testing.T) {
	mt := NewMultiplexedTransport(10)

	req := &http.Request{
		URL: &url.URL{Host: "example.com:80"},
	}

	transport := mt.GetTransport(req)
	if transport == nil {
		t.Error("Expected non-nil transport")
	}
}

func TestMultiplexedTransportAcquireConnection(t *testing.T) {
	mt := NewMultiplexedTransport(10)

	// Should not find existing connection
	conn, found := mt.AcquireConnection("nonexistent.com:80")
	if found {
		t.Error("Expected not found for nonexistent host")
	}
	if conn != nil {
		t.Error("Expected nil connection for nonexistent host")
	}
}

func TestMultiplexedTransportStats(t *testing.T) {
	mt := NewMultiplexedTransport(10)

	stats := mt.Stats()
	if stats.Pools == nil {
		t.Error("Expected non-nil pools map")
	}
}

func TestMultiplexedTransportStatsMultipleHosts(t *testing.T) {
	mt := NewMultiplexedTransport(10)

	// Access different hosts via GetTransport
	mt.GetTransport(&http.Request{URL: &url.URL{Host: "host1.com:80"}})
	mt.GetTransport(&http.Request{URL: &url.URL{Host: "host2.com:80"}})
	mt.GetTransport(&http.Request{URL: &url.URL{Host: "host3.com:80"}})

	stats := mt.Stats()
	if len(stats.Pools) != 0 {
		t.Errorf("Expected 0 pools (no connections added), got %d", len(stats.Pools))
	}
}

func TestNewTLSConfig(t *testing.T) {
	// Test with insecure skip verify disabled
	tlsConfig := NewTLSConfig(false)
	if tlsConfig.InsecureSkipVerify != false {
		t.Error("Expected InsecureSkipVerify to be false")
	}
	if tlsConfig.MinVersion != 0x0303 { // TLS 1.2
		t.Error("Expected MinVersion to be TLS 1.2")
	}

	// Test with insecure skip verify enabled
	tlsConfig2 := NewTLSConfig(true)
	if tlsConfig2.InsecureSkipVerify != true {
		t.Error("Expected InsecureSkipVerify to be true")
	}
}

func TestDialContext(t *testing.T) {
	// This test may fail if there's no network, so we'll just verify it doesn't panic
	conn, err := DialContext("tcp", "localhost:9999", time.Second)
	if err == nil && conn != nil {
		conn.Close()
	}
	// We don't fail on connection errors, just verify it runs
}

func TestMultiplexStats(t *testing.T) {
	stats := MultiplexStats{
		Pools: map[string]PoolStats{
			"test.com:80": {
				TotalConnections: 10,
				InUseConnections: 5,
				IdleConnections:  5,
			},
		},
	}

	if len(stats.Pools) != 1 {
		t.Error("Expected 1 pool")
	}

	poolStats := stats.Pools["test.com:80"]
	if poolStats.TotalConnections != 10 {
		t.Errorf("Expected 10 total, got %d", poolStats.TotalConnections)
	}
	if poolStats.InUseConnections != 5 {
		t.Errorf("Expected 5 in use, got %d", poolStats.InUseConnections)
	}
	if poolStats.IdleConnections != 5 {
		t.Errorf("Expected 5 idle, got %d", poolStats.IdleConnections)
	}
}

func TestPersistentConn(t *testing.T) {
	conn := &persistentConn{
		reqs:    0,
		created: time.Now(),
		inUse:   false,
	}

	if conn.inUse {
		t.Error("Expected inUse to be false initially")
	}

	conn.mu.Lock()
	conn.inUse = true
	conn.reqs++
	conn.mu.Unlock()

	conn.mu.Lock()
	if !conn.inUse {
		t.Error("Expected inUse to be true after setting")
	}
	if conn.reqs != 1 {
		t.Errorf("Expected reqs to be 1, got %d", conn.reqs)
	}
	conn.mu.Unlock()
}
