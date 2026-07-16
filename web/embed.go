// Package web embeds the built frontend so NetInsight ships as a single static
// binary with no external asset dependencies (air-gap friendly).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the embedded frontend build rooted at the dist directory.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
