package sse

import (
	"fmt"
	"net/http"
	"strings"
)

type Stream struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func Start(w http.ResponseWriter) *Stream {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	return &Stream{w: w, flusher: flusher}
}

func (s *Stream) Write(frame string) bool {
	if _, err := fmt.Fprint(s.w, frame); err != nil {
		return false
	}

	if s.flusher != nil {
		s.flusher.Flush()
	}

	return true
}

func (s *Stream) Data(line string) bool {
	return s.Write("data: " + strings.ReplaceAll(line, "\n", " ") + "\n\n")
}
