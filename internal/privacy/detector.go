package privacy

import (
	"net/http"
	"strings"
)

// DetectSourceApp extracts the client app calling the gateway
func DetectSourceApp(r *http.Request) string {
	userAgent := strings.ToLower(r.Header.Get("User-Agent"))

	// Custom application tracing headers
	if srcHeader := r.Header.Get("X-Source-App"); srcHeader != "" {
		return strings.ToLower(srcHeader)
	}

	if strings.Contains(userAgent, "open-webui") || strings.Contains(userAgent, "openwebui") {
		return "openwebui"
	}
	if strings.Contains(userAgent, "openclaude") || strings.Contains(userAgent, "claude-code") {
		return "openclaude"
	}
	if strings.Contains(userAgent, "codex") {
		return "codex"
	}
	if strings.Contains(userAgent, "antigravity") {
		return "antigravity"
	}

	return "direct-api"
}
