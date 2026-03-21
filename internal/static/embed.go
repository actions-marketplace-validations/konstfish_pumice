package static

import "embed"

//go:embed *.css *.js
var Files embed.FS
