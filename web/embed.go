package web

import "embed"

//go:embed index.html app.js app.css components assets
var FS embed.FS
