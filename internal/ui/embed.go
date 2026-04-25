package ui

import "embed"

// FS holds the compiled frontend assets embedded at build time.
// Populate by running `npm run build` in the frontend/ directory.
//
//go:embed all:dist
var FS embed.FS
