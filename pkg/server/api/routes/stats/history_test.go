package stats

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/types"
)

func ratePoints(ts0 int64, rx, tx float64) []types.IfaceHistoryPoint {
	return []types.IfaceHistoryPoint{
		{TS: ts0},
		{TS: ts0 + 1, RxBytesPS: &rx, TxBytesPS: &tx},
	}
}

func TestSummarizeUsageExcludesNonPhysicalFromTotal(t *testing.T) {
	ifaceHistory := map[string][]types.IfaceHistoryPoint{
		"eth0":    ratePoints(0, 100, 200),
		"servers": ratePoints(0, 40, 80),
	}
	physical := map[string]bool{"eth0": true}

	usage := summarizeUsage(ifaceHistory, physical)

	if _, ok := usage.PerIface["servers"]; !ok {
		t.Fatal("vlan interface should still be available per-interface for filtering")
	}

	if usage.Total.RxBytes != 100 || usage.Total.TxBytes != 200 {
		t.Fatalf("total should only count physical eth0, got rx=%d tx=%d", usage.Total.RxBytes, usage.Total.TxBytes)
	}
}
