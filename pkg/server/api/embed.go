//go:build prod

package api

import "embed"

//go:embed all:dist
var EmbeddedFS embed.FS
