package loadbalance

import (
	"testing"
)

func TestPoolCreation(t *testing.T) {
	pool := NewPool(RoundRobin)
	if pool == nil {
		t.Error("Expected non-nil pool")
	}
}

func TestPoolAddServer(t *testing.T) {
	pool := NewPool(RoundRobin)

	server := pool.AddServer("localhost:8080", 1)
	if server == nil {
		t.Error("Expected non-nil server")
	}

	servers := pool.GetServers()
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}
}

func TestPoolRemoveServer(t *testing.T) {
	pool := NewPool(RoundRobin)

	pool.AddServer("localhost:8080", 1)
	pool.AddServer("localhost:8081", 1)

	pool.RemoveServer("localhost:8080")

	servers := pool.GetServers()
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}
}

func TestPoolSelect(t *testing.T) {
	pool := NewPool(RoundRobin)

	pool.AddServer("localhost:8080", 1)
	pool.AddServer("localhost:8081", 1)

	server := pool.Select()
	if server == nil {
		t.Error("Expected non-nil server")
	}

	pool.Release(server)
}

func TestPoolSelectNoServers(t *testing.T) {
	pool := NewPool(RoundRobin)

	server := pool.Select()
	if server != nil {
		t.Error("Expected nil server when pool is empty")
	}
}

func TestPoolSelectWithWeight(t *testing.T) {
	pool := NewPool(Weighted)

	pool.AddServer("localhost:8080", 1)
	pool.AddServer("localhost:8081", 3) // 3x weight

	// Run multiple selections to see distribution
	var server1Count, server2Count int
	for i := 0; i < 100; i++ {
		server := pool.Select()
		if server != nil {
			if server.Address == "localhost:8080" {
				server1Count++
			} else {
				server2Count++
			}
			pool.Release(server)
		}
	}

	// Server with higher weight should be selected more often
	if server2Count <= server1Count {
		t.Errorf("Expected server2 to be selected more often, got server1=%d, server2=%d", server1Count, server2Count)
	}
}

func TestPoolLeastConnections(t *testing.T) {
	pool := NewPool(LeastConnections)

	pool.AddServer("localhost:8080", 1)
	pool.AddServer("localhost:8081", 1)

	// Select first server
	server1 := pool.Select()
	if server1 == nil {
		t.Fatal("Expected non-nil server")
	}

	// Release it
	pool.Release(server1)

	// Select again - should get same server (least connections)
	server2 := pool.Select()
	if server2 == nil {
		t.Fatal("Expected non-nil server")
	}

	// Both should have 0 connections now
	pool.Release(server2)
}

func TestPoolRecordFailure(t *testing.T) {
	pool := NewPool(RoundRobin)

	pool.AddServer("localhost:8080", 1)

	// Record 3 failures
	pool.RecordFailure("localhost:8080")
	pool.RecordFailure("localhost:8080")
	pool.RecordFailure("localhost:8080")

	healthyServers := pool.GetHealthyServers()
	if len(healthyServers) != 0 {
		t.Errorf("Expected 0 healthy servers after 3 failures, got %d", len(healthyServers))
	}
}

func TestPoolRecordSuccess(t *testing.T) {
	pool := NewPool(RoundRobin)

	pool.AddServer("localhost:8080", 1)

	// Record failures first
	pool.RecordFailure("localhost:8080")
	pool.RecordFailure("localhost:8080")
	pool.RecordFailure("localhost:8080")

	// Record success - should become healthy again
	pool.RecordSuccess("localhost:8080")

	healthyServers := pool.GetHealthyServers()
	if len(healthyServers) != 1 {
		t.Errorf("Expected 1 healthy server, got %d", len(healthyServers))
	}
}

func TestPoolUpdateLatency(t *testing.T) {
	pool := NewPool(LeastLatency)

	pool.AddServer("localhost:8080", 1)
	pool.AddServer("localhost:8081", 1)

	pool.UpdateLatency("localhost:8080", 100)
	pool.UpdateLatency("localhost:8081", 50)

	// Select should prefer lower latency server
	server := pool.Select()
	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.Address != "localhost:8081" {
		t.Errorf("Expected localhost:8081 (lower latency), got %s", server.Address)
	}

	pool.Release(server)
}

func TestServerStats(t *testing.T) {
	pool := NewPool(RoundRobin)

	pool.AddServer("localhost:8080", 2)

	server := pool.Select()
	if server != nil {
		pool.Release(server)
	}

	stats := pool.ServerStats()
	if len(stats) != 1 {
		t.Errorf("Expected 1 server stats, got %d", len(stats))
	}

	if stats[0]["address"] != "localhost:8080" {
		t.Errorf("Expected address localhost:8080, got %v", stats[0]["address"])
	}
}

func TestGetHealthyServers(t *testing.T) {
	pool := NewPool(RoundRobin)

	pool.AddServer("localhost:8080", 1)
	pool.AddServer("localhost:8081", 1)
	pool.AddServer("localhost:8082", 1)

	// Mark one as unhealthy
	pool.RecordFailure("localhost:8080")
	pool.RecordFailure("localhost:8080")
	pool.RecordFailure("localhost:8080")

	healthy := pool.GetHealthyServers()
	if len(healthy) != 2 {
		t.Errorf("Expected 2 healthy servers, got %d", len(healthy))
	}
}
