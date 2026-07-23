# Installation

Routier targets Alpine Linux and ships as a signed `apk` package for an
existing box.

## Package install

Routier's dependencies (FRR, Kea, keepalived, radvd, and more) come from
Alpine's community repository, so enable it first:

```bash
setup-apkrepos -c -1
wget -qO /etc/apk/keys/routier.rsa.pub %%REPO_URL%%/routier.rsa.pub
apk add --repository %%REPO_URL%%/nightly routier
```

The package installs:

- `/usr/bin/routier` (setuid root, group `wheel`, mode 4750: the daemon runs
  it as root, `wheel` members can escalate through it, everyone else cannot
  run it),
- OpenRC services `routier` and `routier-ui`,
- a crontab entry for the rollback watchdog,
- the host takeover tool (`/usr/sbin/routier-takeover`).

### Host network takeover

Routier manages interfaces through netlink and service runlevels itself, so
Alpine's native `networking` (ifupdown) service conflicts with it.
`post-install` runs the takeover tool, which:

1. backs up the host's network configuration (`/etc/network/interfaces`,
   dhcpcd, resolv.conf, wpa_supplicant, sysctl, nftables, FRR) to
   `/var/lib/routier/host-backup/<timestamp>/`, with a manifest and a
   generated `restore.sh` for returning to the previous setup;
2. disables native networking from boot **only if** Routier can keep the box
   online: either an existing `/etc/routier/config.yml`, or a DHCP fallback
   config synthesized from the current default-route interface. Otherwise it
   backs up, warns, and leaves native networking enabled.

In that case the host keeps its existing connectivity, so configure Routier
from the web UI instead: start it with `rc-service routier-ui start` (it also
comes up on boot) and browse to port 8080, logging in with the default
`routier` / `admin`. Once you apply a config, `/etc/routier/config.yml`
is written and native networking is handed over; re-run `routier takeover` if
you want to disable the native stack immediately rather than on the next apply.

The takeover is idempotent and never runs on upgrade: `post-upgrade` only
restarts `routier-ui` and touches neither config nor backups.

## First run

Browse to the web UI on port 8080 and log in with the default credentials
`routier` / `admin` (or the one-time password placed in
`/var/lib/routier/ui-seed-password`, which is consumed on first start). A fresh
installation starts with an onboarding wizard:

1. change the admin password,
2. set the hostname,
3. configure interfaces (guided uplink DHCP, optional static LAN),
4. review and apply.

Every step maps to plain config sections, so the wizard can also be skipped by
writing `/etc/routier/config.yml` by hand and running `routier apply`.

## Upgrades

```bash
apk upgrade routier
```

Config, database and backups are never touched by an upgrade. Peer-facing
protocol changes are not negotiated, so friend nodes are expected to run the
same version and upgrade together.
