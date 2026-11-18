# Installation

Routier targets Alpine Linux. It ships as a signed `apk` package for an
existing box, and as installation media that install Alpine and Routier onto
a disk.

## Installation media

The installer ISO is published for `x86_64` and `aarch64`:
[`routier-latest-x86_64.iso`](%%REPO_URL%%/iso/routier-latest-x86_64.iso) and
[`routier-latest-aarch64.iso`](%%REPO_URL%%/iso/routier-latest-aarch64.iso).
Each release is also kept as `routier-installer-<version>-r<release>-<arch>.iso`
in the same directory.

Two kinds of ISO can be built (see [building.md](building.md)): the light
installer ISO that is published, and the appliance ISO, which runs the whole
stack from memory. Both bring every network interface up with DHCP at boot and
start the web UI.

The console shows where to reach it and a one-time password, regenerated on
every live boot:

```
  This system runs from the installation media.

  Install from a browser:
    https://192.0.2.10:8080
  Sign in as routier with the password mB5avGD5ML6xT3FH

  Or install from this console: setup-routier
```

After signing in, the web UI offers to install to a disk. On the appliance
ISO it can also continue without installing, which runs the onboarding wizard
on the live system; changes are lost on restart. The installation steps:

1. pick the target disk (the installation media itself cannot be selected),
2. set the hostname, the timezone and the admin password of the installed
   system,
3. confirm that the disk is erased, then follow the installation log,
4. remove the installation media and restart.

The installed system brings every interface up with DHCP and starts the
onboarding wizard after signing in as `routier` with the chosen password.

On the console, `setup-routier` asks the same questions interactively, plus
the role of each interface. `routier setup --disk /dev/sda --hostname edge
--timezone UTC --password-stdin --no-reboot` installs without prompts.

## Cloud images

`task cloud` builds a UEFI qcow2 image (see [building.md](building.md)) with
Alpine, Routier, cloud-init and sshd. The root filesystem grows to the size of
the instance disk on first boot, and the serial console is enabled.

cloud-init configures the network and the default `alpine` user, which signs
in over SSH with the keys from the user-data and has passwordless `doas`. A
`routier` key in the user-data provisions Routier on the
first boot of each instance:

```yaml
#cloud-config
ssh_authorized_keys:
  - ssh-ed25519 AAAA... admin@example
routier:
  password: a-long-admin-password
  config:
    version: v5.0.0
    hostname: edge
    interfaces:
      wan:
        select: eth[0]
        addresses: [dhcp]
```

- `password` sets the password of the `routier` web UI user. Without it, a
  one-time password is generated and printed on the serial console, and the
  web UI asks to change it at first login.
- `config` is a Routier configuration, as a mapping or as a YAML string. It is
  validated, written to `/etc/routier/config.yml` and applied. Routier then
  takes over networking (see [Host network takeover](#host-network-takeover))
  and cloud-init stops rendering the network
  (`/etc/cloud/cloud.cfg.d/99-routier-network.cfg`). The onboarding wizard is
  skipped.

- `cibi` joins a [cibi](https://github.com/ChevalRouting/cibi) fleet manager
  with a one-time enrollment code from cibi's Routiers page:

  ```yaml
  routier:
    cibi:
      server: https://cibi.example.com
      code: 4BAY-RPCI-R57U-SXRP-KPE7-HEWJ
  ```

  Once the web server is up, Routier creates an API key named
  `cibi <date>`, sends it to cibi with its identity, and cibi connects back
  to check the identity before registering it. `url` sets the address cibi
  reaches the routier at (by default the address cibi sees, on `port`, 8080
  by default), and `insecure: true` accepts any certificate from cibi. A
  failed attempt is retried every 30 seconds for a day; a code cibi refuses
  stops it. The same join runs by hand with
  `routier cibi join https://cibi.example.com <code>`.

Without `config`, cloud-init keeps managing the network and the web UI starts
with the onboarding wizard, as after a package install. The section is applied
once per instance id: later changes made in the web UI are kept across
reboots, while a new instance id (an instance launched from a snapshot, for
example) applies the section again and replaces the configuration.
cloud-init reports `routier` as an unknown key in its schema validation
warning; the warning is harmless.

## Package install

Routier's dependencies (FRR, Kea, keepalived, radvd, and more) come from
Alpine's community repository, so enable it first:

```bash
setup-apkrepos -c -1
wget -qO /etc/apk/keys/routier.rsa.pub %%REPO_URL%%/routier.rsa.pub
echo %%REPO_URL%%/nightly >> /etc/apk/repositories
apk update && apk add routier
```

`nightly` follows every change on the main branch; `stable` holds tagged
releases once they exist. Keeping the channel in `/etc/apk/repositories` lets
`apk upgrade` and the System › General › Updates page install new builds. The
ISOs and cloud images list the nightly channel already.

The package installs:

- `/usr/bin/routier` (setuid root, group `wheel`, mode 4750: the daemon runs
  it as root, `wheel` members can escalate through it, everyone else cannot
  run it),
- OpenRC services `routier`, `routier-ui` and `routier-net`,
- a crontab entry for the rollback watchdog.

Installing does not touch the host's network. `post-install` creates the
`routier` user, generates a one-time password for it, starts the web UI and
prints where to reach it:

```
routier installed. Networking is unchanged until you apply and confirm a configuration.

Web UI:
  https://192.0.2.10:8080

Login: routier / mB5avGD5ML6xT3FH
(one-time password, changed at first login)
```

The password is also kept in `/var/lib/routier/ui-seed-password` until it is
changed. If a `routier` user already exists, its password is left as is.

## First run

Log in to the web UI with the printed credentials. A fresh installation starts
with an onboarding wizard:

1. change the admin password,
2. set the hostname,
3. review the interfaces, pre-filled from the host's current network: the
   interface holding the default route becomes the Internet uplink (DHCP if a
   DHCP client runs on it, otherwise its static address, gateway and DNS
   servers), and other statically addressed interfaces become local networks,
4. review and apply.

Every step maps to plain config sections, so the wizard can also be skipped by
writing `/etc/routier/config.yml` by hand and running `routier apply`.

### Host network takeover

Routier manages interfaces through netlink and service runlevels itself, so
Alpine's native `networking` (ifupdown) and `dhcpcd` services conflict with it.
Until a configuration is confirmed, the native stack stays in charge: an apply
that is not confirmed within 60 seconds rolls back, and the `routier` boot
service does not apply anything while native networking is enabled.

The first confirmed apply hands networking over:

1. the host's network configuration (`/etc/network/interfaces`, dhcpcd,
   resolv.conf, wpa_supplicant, sysctl, nftables, FRR) is backed up to
   `/var/lib/routier/host-backup/<timestamp>/`, with a manifest and a
   generated `restore.sh`, and `host-backup/latest` points to it;
2. the native network services are removed from the boot and default
   runlevels (they keep running until the next reboot), and
   `/etc/network/interfaces` is reduced to the loopback;
3. `/var/lib/routier/taken-over` records that the handover happened.

`routier takeover` runs the same handover by hand. It requires an existing
`/etc/routier/config.yml` unless `--force` is given.

## Removal

```bash
apk del routier routier-openrc
```

`pre-deinstall` stops and removes the Routier services. If the host was taken
over, it runs `host-backup/latest/restore.sh`, which restores the backed up
files and re-enables the native network services; reboot to return to them.
`/etc/routier` and `/var/lib/routier` are kept.

## Upgrades

System › General › Updates in the web UI lists pending package updates and installs
them, including Routier itself. From a shell:

```bash
apk update && apk upgrade routier
```

Config, database and backups are never touched by an upgrade. Peer-facing
protocol changes are not negotiated, so friend nodes are expected to run the
same version and upgrade together.
