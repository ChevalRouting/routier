package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed docs/swagger.json
var openAPISpec []byte

//go:embed docs/swagger-ui/*
var swaggerUIFiles embed.FS

const swaggerUI = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Routier API</title>
  <link rel="stylesheet" href="swagger-ui/swagger-ui.css">
  <style>body { margin: 0; }</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="swagger-ui/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({ url: '/api/v1/docs/openapi.json', dom_id: '#swagger-ui' });
    };
  </script>
</body>
</html>`

func docsRoutes(r chi.Router) {
	r.Get("/api/v1/docs/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(openAPISpec)
	})
	r.Get("/api/v1/docs", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/api/v1/docs/", http.StatusMovedPermanently)
	})
	r.Get("/api/v1/docs/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerUI))
	})

	assets, _ := fs.Sub(swaggerUIFiles, "docs/swagger-ui")
	r.Handle("/api/v1/docs/swagger-ui/*", http.StripPrefix("/api/v1/docs/swagger-ui/", http.FileServer(http.FS(assets))))
}
