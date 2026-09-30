package bondstat

import "testing"

const lacpHealthy = `Ethernet Channel Bonding Driver: v5.15

Bonding Mode: IEEE 802.3ad Dynamic link aggregation
Transmit Hash Policy: layer3+4 (1)
MII Status: up
MII Polling Interval (ms): 100

802.3ad info
LACP rate: fast
Min links: 0
Active Aggregator Info:
	Aggregator ID: 2
	Number of ports: 2

Slave Interface: eth0
MII Status: up
Speed: 10000 Mbps
Duplex: full
Link Failure Count: 0
Aggregator ID: 2
Actor Churn State: none
Partner Churn State: none

Slave Interface: eth1
MII Status: up
Speed: 10000 Mbps
Duplex: full
Link Failure Count: 1
Aggregator ID: 2
Actor Churn State: none
Partner Churn State: none
`

func TestParseLACPHealthy(t *testing.T) {
	s := Parse("bond0", lacpHealthy)

	if !s.LACP {
		t.Fatal("expected LACP mode")
	}

	if s.XmitHashPolicy != "layer3+4" {
		t.Errorf("xmit hash = %q", s.XmitHashPolicy)
	}

	if s.LACPRate != "fast" || s.ActiveAggregator != 2 {
		t.Errorf("rate=%q agg=%d", s.LACPRate, s.ActiveAggregator)
	}

	if len(s.Slaves) != 2 || s.Slaves[0].Speed != 10000 || s.Slaves[1].LinkFailures != 1 {
		t.Fatalf("slaves = %+v", s.Slaves)
	}

	if !s.Healthy {
		t.Error("expected healthy LACP bond")
	}
}

func TestParseLACPChurned(t *testing.T) {
	churned := lacpHealthy[:len(lacpHealthy)-len("Partner Churn State: none\n")] + "Partner Churn State: churned\n"
	s := Parse("bond0", churned)

	if s.Healthy {
		t.Error("bond with a churned partner should be unhealthy")
	}
}

func TestParseSlaveDown(t *testing.T) {
	s := Parse("bond0", lacpHealthy)
	s = Parse("bond0", replaceFirst(lacpHealthy, "Slave Interface: eth0\nMII Status: up", "Slave Interface: eth0\nMII Status: down"))
	if s.Healthy {
		t.Error("bond with a down slave should be unhealthy")
	}
}

func replaceFirst(s, old, new string) string {
	i := indexOf(s, old)
	if i < 0 {
		return s
	}

	return s[:i] + new + s[i+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}

	return -1
}

func TestParseSplitAggregator(t *testing.T) {
	s := Parse("bond0", replaceFirst(lacpHealthy, "Link Failure Count: 1\nAggregator ID: 2", "Link Failure Count: 1\nAggregator ID: 3"))
	if s.Healthy {
		t.Fatal("members in different aggregators must be degraded")
	}
}
