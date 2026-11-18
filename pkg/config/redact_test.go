package config

import "testing"

func TestRedactedStripsSecrets(t *testing.T) {
	cfg := &Config{
		Hostname: "edge",
		Friends:  []*Friend{{Name: "b", Token: "tok"}},
		Users:    map[string]*User{"admin": {PasswordHash: "$6$hash"}},
		VPN: VPNConfig{Wireguard: map[string]*Wireguard{"wg0": {
			PrivateKey: "priv",
			Peers:      []WGPeer{{PublicKey: "pub", PresharedKey: "psk"}},
		}}},
		HA: &HA{VRRP: []*VRRPInstance{{ID: 1, Interface: "lan", Password: "vrrp"}}},
	}

	out, err := Redacted(cfg)
	if err != nil {
		t.Fatal(err)
	}

	wg := out.VPN.Wireguard["wg0"]
	if out.Friends[0].Token != "" || out.Users["admin"].PasswordHash != "" || wg.PrivateKey != "" || wg.Peers[0].PresharedKey != "" {
		t.Fatalf("secrets left in redacted config: %+v", out)
	}

	if wg.Peers[0].PublicKey != "pub" || out.Hostname != "edge" || out.HA.VRRP[0].Password != "vrrp" {
		t.Fatalf("non-secret fields lost: %+v", out)
	}

	if cfg.Friends[0].Token != "tok" || cfg.VPN.Wireguard["wg0"].PrivateKey != "priv" {
		t.Fatal("source config was modified")
	}
}
