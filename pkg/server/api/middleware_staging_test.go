package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSExposesStagingHeaders(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodOptions} {
		t.Run(method, func(t *testing.T) { testCORSStagingHeaders(t, method) })
	}
}

func testCORSStagingHeaders(t *testing.T, method string) {
	t.Helper()
	handler := corsMiddleware(http.HandlerFunc(stagingHeadersTestHandler))
	request := httptest.NewRequest(method, "/api/config/hostname", nil)
	request.Header.Set("Origin", "https://ui.example.test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	exposed := response.Header().Get("Access-Control-Expose-Headers")
	for _, name := range []string{"X-Staging-Pending", "X-Staging-Layer"} {
		if !strings.Contains(exposed, name) {
			t.Fatalf("browser cannot read %s: %q", name, exposed)
		}
	}
}

func stagingHeadersTestHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("X-Staging-Pending", "true")
	w.Header().Set("X-Staging-Layer", "advanced")
	w.WriteHeader(http.StatusOK)
}
