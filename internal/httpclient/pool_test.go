package httpclient

import (
	"testing"
)

func TestConnectionPool_GetTransport(t *testing.T) {
	pool := NewConnectionPool(100)

	transport1 := pool.GetTransport("api.openai.com")
	if transport1 == nil {
		t.Error("Expected non-nil transport")
	}

	// Same host should return same transport
	transport2 := pool.GetTransport("api.openai.com")
	if transport1 != transport2 {
		t.Error("Expected same transport for same host")
	}

	// Different host should return different transport
	transport3 := pool.GetTransport("api.anthropic.com")
	if transport1 == transport3 {
		t.Error("Expected different transport for different host")
	}
}

func TestConnectionPool_Stats(t *testing.T) {
	pool := NewConnectionPool(50)

	// Initially empty
	stats := pool.Stats()
	if stats["total_transports"] != 0 {
		t.Errorf("Expected 0 transports, got %d", stats["total_transports"])
	}

	// After getting transports
	pool.GetTransport("host1.com")
	pool.GetTransport("host2.com")
	pool.GetTransport("host3.com")

	stats = pool.Stats()
	if stats["total_transports"] != 3 {
		t.Errorf("Expected 3 transports, got %d", stats["total_transports"])
	}

	if stats["pool_size"] != 50 {
		t.Errorf("Expected pool size 50, got %d", stats["pool_size"])
	}
}

func TestConnectionPool_Close(t *testing.T) {
	pool := NewConnectionPool(100)

	// Create some transports
	pool.GetTransport("host1.com")
	pool.GetTransport("host2.com")

	stats := pool.Stats()
	if stats["total_transports"] != 2 {
		t.Errorf("Expected 2 transports before close, got %d", stats["total_transports"])
	}

	// Close all
	pool.Close()

	stats = pool.Stats()
	if stats["total_transports"] != 0 {
		t.Errorf("Expected 0 transports after close, got %d", stats["total_transports"])
	}
}

func TestConnectionPool_ConcurrentAccess(t *testing.T) {
	pool := NewConnectionPool(100)

	done := make(chan bool)

	// Concurrent access
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				pool.GetTransport("host.com")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have exactly 1 transport for "host.com"
	stats := pool.Stats()
	if stats["total_transports"] != 1 {
		t.Errorf("Expected 1 transport, got %d", stats["total_transports"])
	}
}

func TestConnectionPool_TransportConfiguration(t *testing.T) {
	pool := NewConnectionPool(50)

	transport := pool.GetTransport("api.example.com")

	// Verify transport configuration
	if transport.MaxIdleConns != 50 {
		t.Errorf("Expected MaxIdleConns=50, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 50 {
		t.Errorf("Expected MaxIdleConnsPerHost=50, got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 90*1e9 { // 90 seconds in nanoseconds
		t.Errorf("Expected IdleConnTimeout=90s")
	}
}
