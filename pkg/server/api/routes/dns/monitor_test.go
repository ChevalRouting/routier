package dns

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQueryStreamConnectsWithoutQueries(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "tail"), []byte("#!/bin/sh\nwhile :; do :; done\n"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &cancelOnFlush{ResponseRecorder: httptest.NewRecorder(), cancel: cancel}
	done := make(chan struct{})
	go func() {
		defer close(done)
		QueryStream(w, httptest.NewRequest("GET", "/api/dns/queries/stream", nil).WithContext(ctx))
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stream did not stop on cancellation")
	}

	if !w.Flushed || !strings.Contains(w.Body.String(), ": connected\n\n") {
		t.Fatalf("missing initial flush: %s", w.Body.String())
	}

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatal("incorrect content type")
	}
}

type cancelOnFlush struct {
	*httptest.ResponseRecorder
	cancel context.CancelFunc
}

func (w *cancelOnFlush) Flush() { w.ResponseRecorder.Flush(); w.cancel() }

func TestQueryStreamStartFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	w := httptest.NewRecorder()
	QueryStream(w, httptest.NewRequest("GET", "/api/dns/queries/stream", nil))
	if w.Code != 500 {
		t.Fatalf("status = %d", w.Code)
	}

	if strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatal("startup failure advertised as stream")
	}
}
