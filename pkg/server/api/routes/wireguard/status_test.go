package wireguard

import "testing"

const dump = "wg-paris\tPRIV\tPUBLOCAL\t51820\toff\n" +
	"wg-paris\tPUBPEER\tPSK\t198.51.100.2:51821\t10.255.0.1/32,10.30.0.0/24\t1792738000\t1024\t2048\t25\n" +
	"wg-lyon\tPRIV2\tPUBLOCAL2\t51821\toff\n" +
	"wg-lyon\tPUBPEER2\t(none)\t(none)\t(none)\t0\t0\t0\toff\n"

func TestParseDump(t *testing.T) {
	status := ParseDump(dump)
	if len(status) != 2 || status[0].Interface != "wg-paris" || status[0].ListenPort != 51820 || len(status[0].Peers) != 1 {
		t.Fatalf("status %+v", status)
	}

	peer := status[0].Peers[0]
	if peer.Endpoint != "198.51.100.2:51821" || peer.LatestHandshake != 1792738000 || peer.RxBytes != 1024 || peer.TxBytes != 2048 || len(peer.AllowedIPs) != 2 {
		t.Fatalf("peer %+v", peer)
	}

	idle := status[1].Peers[0]
	if idle.Endpoint != "" || idle.LatestHandshake != 0 || len(idle.AllowedIPs) != 0 {
		t.Fatalf("idle peer %+v", idle)
	}
}
