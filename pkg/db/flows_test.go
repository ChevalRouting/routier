package db

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/types"
)

func sampleFlow(end int64, src, dst string, bytes int64) types.Flow {
	return types.Flow{Start: end - 10, End: end, InIf: "wan", OutIf: "lan", Family: 4, Proto: 6, Src: src, Dst: dst, SrcPort: 40000, DstPort: 443, Packets: bytes / 100, Bytes: bytes, Samples: 1}
}

func TestFlowsTopAndRecent(t *testing.T) {
	db := testDB(t)
	flows := []types.Flow{
		sampleFlow(100, "192.0.2.1", "198.51.100.1", 1000),
		sampleFlow(110, "192.0.2.1", "198.51.100.2", 3000),
		sampleFlow(120, "192.0.2.2", "198.51.100.1", 500),
	}

	if err := InsertFlows(t.Context(), db, flows); err != nil {
		t.Fatal(err)
	}

	top, err := TopFlows(t.Context(), db, "src", 0, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(top) != 2 || top[0].Key != "192.0.2.1" || top[0].Bytes != 4000 || top[0].Flows != 2 {
		t.Fatalf("top sources = %+v", top)
	}

	services, err := TopFlows(t.Context(), db, "service", 0, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(services) != 1 || services[0].Key != "6/443" {
		t.Fatalf("top services = %+v", services)
	}

	recent, err := RecentFlows(t.Context(), db, 105, "198.51.100.1", 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(recent) != 1 || recent[0].Src != "192.0.2.2" {
		t.Fatalf("recent flows = %+v", recent)
	}
}

func TestFlowSeriesBuckets(t *testing.T) {
	db := testDB(t)
	if err := InsertFlows(t.Context(), db, []types.Flow{sampleFlow(100, "a", "b", 10), sampleFlow(110, "a", "b", 20), sampleFlow(170, "a", "b", 40)}); err != nil {
		t.Fatal(err)
	}

	points, err := FlowSeries(t.Context(), db, 0, 60)
	if err != nil {
		t.Fatal(err)
	}

	if len(points) != 2 || points[0].Bytes != 30 || points[1].Bytes != 40 {
		t.Fatalf("series = %+v", points)
	}
}

func TestPruneFlowsByAgeAndCount(t *testing.T) {
	db := testDB(t)
	var flows []types.Flow
	for end := int64(1); end <= 10; end++ {
		flows = append(flows, sampleFlow(end*100, "a", "b", 1))
	}

	if err := InsertFlows(t.Context(), db, flows); err != nil {
		t.Fatal(err)
	}

	if err := PruneFlows(t.Context(), db, 300, 5); err != nil {
		t.Fatal(err)
	}

	recent, err := RecentFlows(t.Context(), db, 0, "", 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(recent) != 5 || recent[len(recent)-1].End != 600 {
		t.Fatalf("after prune: %d flows, oldest end %d", len(recent), recent[len(recent)-1].End)
	}
}

func TestFlowPathsGroupsByDimensions(t *testing.T) {
	db := testDB(t)
	flows := []types.Flow{
		sampleFlow(100, "192.0.2.1", "198.51.100.1", 1000),
		sampleFlow(110, "192.0.2.1", "198.51.100.1", 3000),
		sampleFlow(120, "192.0.2.2", "198.51.100.1", 500),
	}

	if err := InsertFlows(t.Context(), db, flows); err != nil {
		t.Fatal(err)
	}

	paths, err := FlowPaths(t.Context(), db, []string{"src", "service", "dst"}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(paths) != 2 || paths[0].Bytes != 4000 || strings.Join(paths[0].Keys, " ") != "192.0.2.1 6/443 198.51.100.1" {
		t.Fatalf("paths = %+v", paths)
	}

	pairs, err := FlowPaths(t.Context(), db, []string{"in_if", "out_if"}, 105, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(pairs) != 1 || len(pairs[0].Keys) != 2 || pairs[0].Keys[0] != "wan" || pairs[0].Bytes != 3500 {
		t.Fatalf("interface pairs = %+v", pairs)
	}

	total, err := FlowBytesSince(t.Context(), db, 105)
	if err != nil || total != 3500 {
		t.Fatalf("total = %d, %v", total, err)
	}
}

type savedName struct {
	address, name string
	at            int64
}

func TestFlowNamesSkipEmptyAndPrune(t *testing.T) {
	db := testDB(t)
	for _, n := range []savedName{{"192.0.2.1", "host.example.", 100}, {"192.0.2.2", "", 100}, {"192.0.2.3", "old.example.", 10}} {
		if err := SaveFlowName(t.Context(), db, n.address, n.name, n.at); err != nil {
			t.Fatal(err)
		}
	}

	names, err := FlowNames(t.Context(), db, []string{"192.0.2.1", "192.0.2.2", "192.0.2.3", "198.51.100.1"})
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 2 || names["192.0.2.1"] != "host.example." {
		t.Fatalf("names = %+v", names)
	}

	if err := PruneFlows(t.Context(), db, 50, 100); err != nil {
		t.Fatal(err)
	}

	names, err = FlowNames(t.Context(), db, []string{"192.0.2.3"})
	if err != nil || len(names) != 0 {
		t.Fatalf("after prune: %+v, %v", names, err)
	}
}
