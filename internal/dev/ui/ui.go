package ui

import "embed"

//go:embed "index.html" "node_modules" "package-lock.json"
var Files embed.FS
