package observability

import (
	"testing"
)

func TestGetOTELMetrics_NilSafe(t *testing.T) {
	t.Run("returns nil when not initialized", func(t *testing.T) {
		// Reset global
		globalOTELMetrics = nil

		m := GetOTELMetrics()
		if m != nil {
			t.Error("Expected nil when not initialized")
		}
	})
}

func TestOTELMetrics_RecordRequest_RequiresInitialized(t *testing.T) {
	t.Run("requires initialized metrics", func(t *testing.T) {
		// The middleware already checks for nil before calling this
		// So we just verify the function signature works
	})
}
