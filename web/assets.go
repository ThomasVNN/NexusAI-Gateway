// Package web provides embedded frontend assets for NexusAI-Gateway.
package web

import "embed"

// Assets is the embedded file system containing frontend build assets.
//
//go:embed dist/*
var Assets embed.FS
