package web

import "embed"

// Assets contains the compiled single-page application distribution files
//
//go:embed all:dist
var Assets embed.FS
