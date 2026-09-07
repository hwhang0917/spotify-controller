// Package web embeds the built guest UI (Vue + Vite + Tailwind).
// Run `npm run build` in this directory before `go build`.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist is the built guest UI rooted at index.html.
var Dist, _ = fs.Sub(dist, "dist")
