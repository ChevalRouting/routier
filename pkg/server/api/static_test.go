package api

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

type staticCacheCase struct {
	path   string
	status int
	cache  string
}

func TestStaticCachePolicy(t *testing.T) {
	files := fstest.MapFS{
		"index.html":     &fstest.MapFile{Data: []byte("app")},
		"assets/page.js": &fstest.MapFile{Data: []byte("export default {}")},
	}

	cases := []staticCacheCase{
		{path: "/debug", status: http.StatusOK, cache: "no-cache, no-store, must-revalidate"},
		{path: "/", status: http.StatusOK, cache: "no-cache, no-store, must-revalidate"},
		{path: "/assets/missing.js", status: http.StatusNotFound, cache: "no-store"},
		{path: "/assets/page.js", status: http.StatusOK, cache: "public, max-age=31536000, immutable"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			checkStaticCachePolicy(t, files, tc)
		})
	}
}

func checkStaticCachePolicy(t *testing.T, files fstest.MapFS, tc staticCacheCase) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, tc.path, nil)
	serveStatic(fs.FS(files), http.FileServer(http.FS(files)), response, request)
	if response.Code != tc.status {
		t.Fatalf("status = %d, want %d", response.Code, tc.status)
	}

	if got := response.Header().Get("Cache-Control"); got != tc.cache {
		t.Fatalf("cache = %q, want %q", got, tc.cache)
	}
}
