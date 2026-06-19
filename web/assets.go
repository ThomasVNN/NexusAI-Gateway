// Package web provides embedded static assets for the gateway dashboard.
// This is a placeholder - actual assets are built during Docker image creation.
package web

import "embed"

// Assets is the embedded filesystem containing static assets.
// The actual assets are copied from the web-builder stage during Docker build.
var Assets embed.FS
