package runtime

import (
	"context"
)

type contextKey string

const runtimeContextKey contextKey = "nexusai-runtime-context"

// Context represents the request context flowing through the gateway's runtime pipeline.
type Context struct {
	RequestID    string                 `json:"request_id"`
	TenantID     string                 `json:"tenant_id"`
	UserIdentity string                 `json:"user_identity"`
	TraceID      string                 `json:"trace_id"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// NewContext creates a new Context.
func NewContext(requestID, tenantID, userIdentity, traceID string, metadata map[string]interface{}) *Context {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	return &Context{
		RequestID:    requestID,
		TenantID:     tenantID,
		UserIdentity: userIdentity,
		TraceID:      traceID,
		Metadata:     metadata,
	}
}

// WithRuntimeContext returns a new context.Context that contains the runtime Context.
func WithRuntimeContext(ctx context.Context, rtCtx *Context) context.Context {
	return context.WithValue(ctx, runtimeContextKey, rtCtx)
}

// FromContext retrieves the runtime Context from context.Context.
func FromContext(ctx context.Context) (*Context, bool) {
	rtCtx, ok := ctx.Value(runtimeContextKey).(*Context)
	return rtCtx, ok
}
