# NX-24: Provider Health & Rotation

## Status: DEPENDENCY PENDING (Draft)

**Backlog Item ID:** 22222222-2222-2222-2222-222222222206
**Depends on:** NX-204 (Provider Registry) — [PR #223](https://github.com/ThomasVNN/NexusAI-Gateway/pull/223)
**Branch:** `feature/NX-24-provider-health-rotation`

---

## Dependency Status

| Dependency | PR | Status | Notes |
|------------|-----|--------|-------|
| NX-204 Provider Registry | [#223](https://github.com/ThomasVNN/NexusAI-Gateway/pull/223) | OPEN | Must merge first |

---

## Overview

This feature implements provider health checks with circuit breaker pattern and automatic provider rotation when the primary provider fails.

### Features

1. **Health Check per Provider**
   - Periodic health checks against provider endpoints
   - Track: latency (p99), error rate, success rate
   - Configurable check interval (default: 30s)

2. **Circuit Breaker**
   - 5 failures in 30s = unhealthy
   - Automatic recovery after timeout
   - Half-open state for testing recovery

3. **Provider Rotation**
   - Automatic switch to next healthy provider
   - Priority-based selection
   - Fallback chain configuration

---

## Implementation Plan

### Phase 1: Health Checker (`internal/provider/health.go`)

```go
// HealthChecker performs periodic health checks on providers
type HealthChecker struct {
    // Configuration
    checkInterval time.Duration
    timeout       time.Duration
    retries       int

    // State
    mu       sync.RWMutex
    status   map[string]*ProviderHealthStatus
    circuit  map[string]*CircuitBreaker
}

// ProviderHealthStatus tracks health metrics for a provider
type ProviderHealthStatus struct {
    ProviderID     string
    IsHealthy      bool
    LastCheck      time.Time
    LatencyP99     time.Duration
    ErrorRate      float64
    SuccessRate    float64
    FailureCount   int
    CircuitState   string
}
```

### Phase 2: Provider Rotation (`internal/provider/rotation.go`)

```go
// ProviderSelector selects the best available provider
type ProviderSelector struct {
    healthChecker *HealthChecker
    strategy     RotationStrategy
}

// RotationStrategy defines how to select providers
type RotationStrategy int

const (
    PriorityBased RotationStrategy = iota
    RoundRobin
    LeastLoaded
    WeightedRandom
)
```

### Phase 3: Integration

- Extend `/v1/providers` routes with health status
- Add `/v1/providers/{id}/health` endpoint
- Add `/v1/providers/select` for routing decisions
- Background scheduler for health checks

---

## Acceptance Criteria

- [ ] Provider rotation working
- [ ] Health checks running
- [ ] Auto-switch tested
- [ ] Circuit breaker triggers after 5 failures
- [ ] Provider recovers after timeout

---

## Configuration

```yaml
health:
  check_interval: 30s
  timeout: 10s
  retries: 2

circuit_breaker:
  failure_threshold: 5
  success_threshold: 3
  timeout: 30s
```

---

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/provider/health.go` | Create | Health checker with circuit breaker |
| `internal/provider/rotation.go` | Create | Provider rotation logic |
| `internal/provider/models.go` | Create | Health status models |
| `internal/gateway/http/handler/provider.go` | Create | HTTP handlers for health endpoints |
| `internal/gateway/http/router/router.go` | Modify | Register provider routes |
| `internal/db/postgres/connection.go` | Modify | Add provider_health table |

---

## Test Plan

1. Unit tests for HealthChecker
2. Unit tests for ProviderSelector
3. Integration test for circuit breaker
4. Integration test for provider rotation
5. E2E test for auto-switch scenario
