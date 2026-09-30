package lldp

import "testing"

const sampleJSON = `{
  "lldp": {
    "interface": [
      {
        "eth0": {
          "via": "LLDP",
          "age": "0 day, 00:01:30",
          "chassis": {
            "core-sw01": {
              "id": { "type": "mac", "value": "00:11:22:33:44:55" },
              "descr": "Cisco IOS Software",
              "mgmt-ip": ["10.0.0.1"],
              "capability": [
                { "type": "Bridge", "enabled": true },
                { "type": "Router", "enabled": true },
                { "type": "Wlan", "enabled": false }
              ]
            }
          },
          "port": {
            "id": { "type": "ifname", "value": "GigabitEthernet1/0/24" },
            "descr": "uplink to core"
          },
          "vlan": { "vlan-id": "10", "pvid": true, "value": "default" }
        }
      },
      {
        "eth1": {
          "via": "CDP",
          "age": "0 day, 00:00:20",
          "chassis": {
            "switch2.example.com": {
              "id": { "type": "local", "value": "switch2" },
              "mgmt-ip": "192.168.1.2"
            }
          },
          "port": {
            "id": { "type": "ifname", "value": "Fa0/1" }
          }
        }
      }
    ]
  }
}`

func TestShowNeighbors(t *testing.T) {
	restore := SetCommandRunner(func(_ string, _ ...string) ([]byte, error) {
		return []byte(sampleJSON), nil
	})
	defer restore()

	byIface := map[string]Neighbor{}
	for _, n := range ShowNeighbors() {
		byIface[n.LocalIface] = n
	}

	if len(byIface) != 2 {
		t.Fatalf("got %d neighbors, want 2", len(byIface))
	}

	eth0 := byIface["eth0"]
	if eth0.Protocol != "LLDP" {
		t.Errorf("eth0 protocol = %q, want LLDP", eth0.Protocol)
	}
	if eth0.ChassisName != "core-sw01" {
		t.Errorf("eth0 chassis name = %q, want core-sw01", eth0.ChassisName)
	}
	if eth0.ChassisID != "00:11:22:33:44:55" {
		t.Errorf("eth0 chassis id = %q", eth0.ChassisID)
	}
	if eth0.MgmtIP != "10.0.0.1" {
		t.Errorf("eth0 mgmt ip = %q, want 10.0.0.1", eth0.MgmtIP)
	}
	if eth0.PortID != "GigabitEthernet1/0/24" {
		t.Errorf("eth0 port id = %q", eth0.PortID)
	}
	if eth0.Capabilities != "Bridge, Router" {
		t.Errorf("eth0 capabilities = %q, want \"Bridge, Router\"", eth0.Capabilities)
	}
	if eth0.VLAN != "10" {
		t.Errorf("eth0 vlan = %q, want 10", eth0.VLAN)
	}

	eth1 := byIface["eth1"]
	if eth1.Protocol != "CDP" {
		t.Errorf("eth1 protocol = %q, want CDP", eth1.Protocol)
	}
	if eth1.MgmtIP != "192.168.1.2" {
		t.Errorf("eth1 mgmt ip = %q, want 192.168.1.2", eth1.MgmtIP)
	}
	if eth1.PortID != "Fa0/1" {
		t.Errorf("eth1 port id = %q, want Fa0/1", eth1.PortID)
	}
}

func TestShowNeighborsEmpty(t *testing.T) {
	restore := SetCommandRunner(func(_ string, _ ...string) ([]byte, error) {
		return nil, nil
	})
	defer restore()

	if got := ShowNeighbors(); got != nil {
		t.Errorf("ShowNeighbors() = %v, want nil", got)
	}
}
