package web

import "embed"

//go:embed index.html app.js app.css theme-init.js components assets
var FS embed.FS
