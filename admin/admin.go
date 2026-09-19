// Package admin serves the embedded NanoMDM administration console.
package admin

import (
	"embed"
	"net/http"
)

//go:embed static
var assets embed.FS

// Handler returns the HTTP handler for the /admin/ console.
func Handler() http.Handler {
	return http.StripPrefix("/admin/", http.FileServer(http.FS(assets)))
}
