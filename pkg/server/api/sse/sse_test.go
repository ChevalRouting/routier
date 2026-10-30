package sse

import (
	"net/http/httptest"
	"testing"
)

func TestStreamWritesFlushedDataFrames(t *testing.T) {
	rec := httptest.NewRecorder()
	stream := Start(rec)
	if !stream.Data("first\nsecond") || !stream.Write(": ping\n\n") {
		t.Fatal("write failed")
	}

	if got := rec.Body.String(); got != "data: first second\n\n: ping\n\n" {
		t.Fatalf("body = %q", got)
	}

	if rec.Header().Get("Content-Type") != "text/event-stream" || !rec.Flushed {
		t.Fatalf("headers %v, flushed %v", rec.Header(), rec.Flushed)
	}
}
