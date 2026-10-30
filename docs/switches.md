# Switches

Routier can manage the switches hanging off it. Switches are declared in the `switches`
section of the config: their port VLANs and bonds follow the same staging, diff, apply and
watchdog cycle as the rest of the router. Credentials are the exception: they are not part of
the config and are kept encrypted in the routier database.

The first supported family is MikroTik RouterOS 7 (CRS3xx/CRS5xx and other devices with bridge
VLAN filtering), driven through its REST API.

## Configuration

```yaml
switches:
  crs310:
    model: mikrotik-crs310-8g-2s
    host: 10.0.0.2
    port: 443
    tls_skip_verify: true
    interface: lan
    uplink: sfp-sfpplus1
    ports:
      ether1: { mode: access, vlan: 20 }
      ether2: { mode: trunk, native: 10, tagged: [20, 30] }
    bonds:
      bond2:
        members: [ether3, ether4]
        mode: 802.3ad
        port: { mode: trunk, tagged: [10] }
```

| Key | Meaning |
|-----|---------|
| `model` | switch model id (see [Models](#models)) |
| `host` | management address of the switch |
| `scheme` | `https` (default) or `http` to reach the switch API |
| `port` | port of the switch API, `443` for HTTPS and `80` for HTTP by default |
| `tls_skip_verify` | accept a self-signed certificate (HTTPS only) |
| `interface` | routier interface the switch is connected to, for reference |
| `uplink` | switch port facing this routier |
| `ports.<name>.mode` | `access` (untagged, `vlan`) or `trunk` (`tagged`, optional untagged `native`) |
| `bonds.<name>` | bond of at least two `members`, `mode` (`802.3ad` by default) and optional `port` settings |

Only the ports and bonds listed in the config are managed; other ports keep whatever the switch
has. Removing a port from the config leaves it as it is on the switch. Removing a bond from the
config deletes it from the switch, and its members go back to the bridge with the VLANs of the
bond. Changing the members or the mode of a bond deletes and recreates it in the same apply. A
port or a new bond that claims a member of another bond releases that bond first, even one the
config never managed; its other members go back to the bridge with its VLANs. Every bond
deletion runs before the bonds are created, so one apply can break a bond and reuse its ports.

Ports are named after the faceplate, which uses the RouterOS default names (`default-name`).
A port renamed on the switch keeps its default name in the config and in the UI, which also
shows the name set on the switch; changes are sent to the switch under its current name.

The uplink is managed like any other port. A mistake there cuts the routier off from the switch,
and the rollback cannot reach it either: the switch has to be fixed by hand.

Switch changes require bridge VLAN filtering (`vlan-filtering=yes`) on the switch.

## Credentials

Credentials are stored per switch name, sealed with AES-256-GCM in `/var/lib/routier/web.db`.
The key is `/var/lib/routier/vault.key`, created on first use with mode 0600. They never appear
in `config.yml`, in staging, in snapshots or in failure bundles. Set them through the API or the
Switches page before applying a config that declares the switch. Back up `vault.key` together
with the database; copying `config.yml` to another router requires entering the switch
credentials again.

On the switch, enable the REST API and create a dedicated user. HTTPS (`www-ssl`) is the default;
`scheme: http` uses the plain `www` service instead, which sends the credentials unencrypted and
belongs on a trusted management network only.

```
/ip/service enable www-ssl
/user/group add name=routier policy=read,write,api,rest-api,!ftp,!reboot,!policy,!password,!sniff,!sensitive,!romon
/user add name=routier group=routier password=...
```

## Apply and rollback

Each switch renders a desired state artifact, `switches/<name>.json` in
`/etc/routier/out/switches/`. It appears in `routier plan`, in the render diff of the web UI and
in snapshots, and a change to it arms the watchdog like any other artifact.

When the artifact of a switch changes, apply reads the switch, computes the changes against its
live state and the previously applied definition, and runs them. Before the first change, the
live state of the switch is saved next to the snapshot (`/var/lib/routier/switch-images/`). A
failing step fails the apply, which rolls the router back.

Rollback, whether triggered by a failure or by the watchdog, pushes the restored definition to
the switch. Ports and bonds that only the rolled-back definition managed go back to the state
saved before the apply, so a newly added switch, port or bond is undone as well. A rollback that
cannot reach the switch is logged and leaves the switch to be checked by hand.

For RouterOS the changes are grouped in phases: bridge VLAN memberships are narrowed, bridge
ports deleted, bonds deleted or created, bridge ports added, PVIDs and frame types set, then
memberships widened. A bridge VLAN entry covering several IDs (`vlan-ids=100-102`) is split when
only one of them changes. Dynamic entries are left to RouterOS.

At boot, switch errors are reported as warnings and do not stop the boot.

## API

Switch definitions are edited like any other section (`GET/PUT /api/config/switches`, or
`/api/v1/sessions/{id}/switches/{name}`). The switch routes cover what is not configuration:

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/switch-models` | known models with their credential fields and faceplate |
| GET | `/api/switches` | applied switches with model, credentials status and last read state |
| GET | `/api/switches/{name}` | one applied switch |
| POST | `/api/switches/{name}/refresh` | read the switch now |
| GET | `/api/switches/candidates` | LLDP/CDP neighbors matched to models and not configured yet |
| GET | `/api/switches/{name}/plan` | device calls the staged definition would make |
| PUT | `/api/switches/{name}/credentials` | store credentials (`{"values": {"username": ..., "password": ...}}`) |
| DELETE | `/api/switches/{name}/credentials` | forget credentials |

The routier reads every applied switch every minute (`monitoring.collection.switches` in
seconds) and keeps the last state in memory. A switch whose artifact was written since its
last read, by an apply or a rollback, is read again on the next request for its state.

## Models

A model is a YAML file. Built-in models are embedded in routier; files in
`/etc/routier/switch-models/` add models or replace a built-in model with the same `id`.

```yaml
id: mikrotik-crs310-8g-2s
vendor: mikrotik
name: CRS310-8G+2S+
driver: routeros-rest
match:
  sys_descr: 'CRS310-8G\+2S\+'
credentials:
  - { key: username, label: Username, type: text, default: admin, required: true }
  - { key: password, label: Password, type: password, required: true }
faceplate:
  - { group: "2.5GBASE-T", ports: "ether[1-8]", kind: rj45, rows: 1 }
  - { group: "10G SFP+", ports: "sfp-sfpplus[1-2]", kind: sfp-plus, rows: 1 }
```

| Field | Meaning |
|-------|---------|
| `driver` | driver that talks to the device |
| `generic` | fallback model, matched only when no specific model matches |
| `match.sys_descr` | regular expression matched against the LLDP/CDP system description |
| `credentials` | secret fields asked when the switch is enrolled |
| `faceplate` | port groups drawn left to right, labelled by `group` (the port media, such as `1GBASE-T` or `10G SFP+`); `ports` accepts `name[1-8]` ranges and lists |
| `faceplate[].rows` | rows in the group; `order: odd-top` puts odd ports on the top row |

Built-in models: CRS310-8G+2S+, CRS309-1G-8S+, and `mikrotik-generic` for any other RouterOS 7
device (ports are not checked against a faceplate).

## Lab

The development lab (`task dev-up`) includes `fake-switch`, a stateful imitation of the RouterOS
REST API (`dev/fake-routeros`) seeded with a CRS310. It accepts the `routier`/`routier`
credentials and refuses out-of-order changes the way RouterOS does, which makes it possible to
exercise apply, confirm and rollback without hardware.

## Adding a vendor

1. Add a package under `pkg/switches/driver/<name>` implementing `driver.Driver`: `State`
   translates the device configuration into `types.SwitchState`, and `Plan` turns a
   `types.SwitchPlan` into ordered steps that describe and run themselves.
2. Register it in `switches.DefaultDrivers`.
3. Add model files under `pkg/switches/model/catalog/<vendor>/` with `driver: <name>`.
4. Test the driver against recorded API responses, as `pkg/switches/driver/routeros` does.
