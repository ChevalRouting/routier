package stats

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/telemetry/stats"
)

func TestLiveRejectsInvalidInterfaces(t *testing.T) {
	for query, want := range map[string]int{
		"":                 http.StatusBadRequest,
		"?iface=..":        http.StatusBadRequest,
		"?iface=a/b":       http.StatusBadRequest,
		"?iface=routier-x": http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		Live(rec, httptest.NewRequest(http.MethodGet, "/api/stats/live"+query, nil))
		if rec.Code != want {
			t.Fatalf("%q status = %d, want %d", query, rec.Code, want)
		}
	}
}

func TestLiveStreamsSamples(t *testing.T) {
	if _, err := os.Stat("/sys/class/net/lo/statistics/rx_bytes"); err != nil {
		t.Skip("no sysfs interface counters")
	}

	server := httptest.NewServer(http.HandlerFunc(Live))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"?iface=lo", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func(action func() error) { _ = action() }(resp.Body.Close)

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type = %q", ct)
	}

	samples := readLiveSamples(t, bufio.NewScanner(resp.Body), 2)
	if gap := samples[1].TS - samples[0].TS; gap < 900 || gap > 1100 {
		t.Fatalf("sample gap = %d ms", gap)
	}

	if samples[1].RxBytes < samples[0].RxBytes || samples[1].OperState == "" {
		t.Fatalf("samples = %+v", samples)
	}
}

func readLiveSamples(t *testing.T, scanner *bufio.Scanner, count int) []stats.LiveSample {
	var samples []stats.LiveSample
	for len(samples) < count && scanner.Scan() {
		data, found := strings.CutPrefix(scanner.Text(), "data: ")
		if !found {
			continue
		}

		var sample stats.LiveSample
		if err := json.Unmarshal([]byte(data), &sample); err != nil {
			t.Fatal(err)
		}

		samples = append(samples, sample)
	}

	if len(samples) < count {
		t.Fatalf("received %d samples, want %d", len(samples), count)
	}

	return samples
}
