package config

import "testing"

func testVNetConfig() *Config {
	return &Config{Hostname: "r", Interfaces: map[string]*Interface{
		"fabric":  {Type: "vxlan-bridge", MTU: 1450, VXLAN: &VXLAN{Local: "192.0.2.10"}},
		"servers": {Type: "vnet", Select: "fabric", VNet: &VNet{VNI: 10200, VLAN: 200}, Addresses: []string{"10.20.0.1/24"}},
	}}
}

func TestVNetResolveAndValidate(t *testing.T) {
	cfg := testVNetConfig()
	cfg.Interfaces["consumer"] = &Interface{Type: "bridge", Bridge: &Bridge{Members: []string{"servers"}}}
	if errs := Validate(cfg, true); len(errs) != 0 {
		t.Fatal(errs)
	}

	if cfg.Interfaces["servers"].Device != "servers" || cfg.Interfaces["consumer"].Bridge.MemberDevices[0] != "servers" {
		t.Fatal("vnet logical references must resolve to the vnet bridge")
	}

	vx := InterfaceVXLAN(cfg.Interfaces["fabric"])
	if !vx.External || vx.Learning == nil || *vx.Learning {
		t.Fatal("meta-interface must produce an SVD with learning disabled")
	}

	if cfg.Interfaces["fabric"].VXLAN.External {
		t.Fatal("normalization must not mutate saved settings")
	}
}

func TestVNetDuplicateMappings(t *testing.T) {
	cfg := testVNetConfig()
	cfg.Interfaces["other"] = &Interface{Type: "vnet", Select: "fabric", VNet: &VNet{VNI: 10200, VLAN: 200}}
	errs := Validate(cfg, false)
	if !hasErr(errs, "vlan already used") || !hasErr(errs, "vni already used") {
		t.Fatal(errs)
	}
}

func TestVNetInvalidParentAndMapping(t *testing.T) {
	cfg := testVNetConfig()
	cfg.Interfaces["servers"].Select = "missing"
	cfg.Interfaces["servers"].VNet = &VNet{VNI: 16777216, VLAN: 4095}
	errs := Validate(cfg, false)
	if !hasErr(errs, "parent must be a vxlan-bridge") || !hasErr(errs, "vni must") || !hasErr(errs, "vlan must") {
		t.Fatal(errs)
	}
}

func TestVNetPortAndGeneratedCollision(t *testing.T) {
	cfg := testVNetConfig()
	cfg.Interfaces["vx"] = &Interface{Type: "vxlan", VXLAN: &VXLAN{VNI: 1}}
	cfg.Interfaces[VNetLinkName("servers", "a")] = &Interface{Type: "dummy"}
	errs := Validate(cfg, false)
	if !hasErr(errs, "cannot share the port") || !hasErr(errs, "generated device") {
		t.Fatal(errs)
	}
}

func TestL3VNIValidate(t *testing.T) {
	cfg := testVNetConfig()
	cfg.VRFs = map[string]*VRFConfig{"blue": {Table: 100, L3VNI: &VRFL3VNI{VNI: 5000, Transport: "fabric", VLAN: 3999}}}
	cfg.Interfaces["servers"].VRF = "blue"
	if errs := Validate(cfg, false); len(errs) != 0 {
		t.Fatal(errs)
	}
}

func TestL3VNIConflicts(t *testing.T) {
	cfg := testVNetConfig()
	cfg.VRFs = map[string]*VRFConfig{
		"blue":  {Table: 100, L3VNI: &VRFL3VNI{VNI: 10200, Transport: "fabric", VLAN: 200}},
		"green": {Table: 200, L3VNI: &VRFL3VNI{VNI: 0, Transport: "servers", VLAN: 5000}},
	}
	errs := Validate(cfg, false)
	for _, want := range []string{"vrfs.blue.l3vni: vni already used by servers", "vrfs.blue.l3vni: vlan already used by servers", "vrfs.green.l3vni: vni must", "vrfs.green.l3vni: vlan must", "vrfs.green.l3vni: transport must be a vxlan-bridge"} {
		if !hasErr(errs, want) {
			t.Fatalf("missing %q in %v", want, errs)
		}
	}
}

func TestTrackInterfacesStayLogical(t *testing.T) {
	cfg := &Config{
		Interfaces: map[string]*Interface{"wan": {Select: "lo"}},
		HA:         &HA{VRRP: []*VRRPInstance{{ID: 1, Interface: "wan", VIPs: []string{"192.0.2.1/24"}, TrackInterfaces: []string{"wan"}}}},
	}

	ResolveInterfaces(cfg)
	vrrp := cfg.HA.VRRP[0]
	if len(vrrp.TrackInterfaces) != 1 || vrrp.TrackInterfaces[0] != "wan" || len(vrrp.TrackDevices) != 1 || vrrp.TrackDevices[0] != "lo" {
		t.Fatalf("track %+v devices %+v", vrrp.TrackInterfaces, vrrp.TrackDevices)
	}
}

func TestVNetGateway(t *testing.T) {
	cfg := testVNetConfig()
	cfg.Interfaces["servers"].VNet.Gateway = &VNetGateway{IPs: []string{"10.20.0.254", "fd20::1"}}
	if errs := Validate(cfg, false); len(errs) != 0 {
		t.Fatal(errs)
	}

	if mac := VNetGatewayMAC(cfg.Interfaces["servers"].VNet); mac != "02:cb:00:27:d8:01" {
		t.Fatalf("default gateway mac %s", mac)
	}

	cfg.Interfaces["servers"].VNet.Gateway = &VNetGateway{IPs: []string{"10.20.0.254/24", "224.0.0.1"}, MAC: "01:00:5e:00:00:01"}
	errs := Validate(cfg, false)
	if !hasErr(errs, `"10.20.0.254/24" is not a host address`) || !hasErr(errs, `"224.0.0.1" is not a host address`) || !hasErr(errs, "must be a unicast ethernet address") {
		t.Fatal(errs)
	}

	cfg.Interfaces["servers"].VNet.Gateway = &VNetGateway{}
	if !hasErr(Validate(cfg, false), "at least one ip") {
		t.Fatal("a gateway without addresses was accepted")
	}
}
