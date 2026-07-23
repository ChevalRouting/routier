# Routier documentation

Routier turns a plain Alpine Linux box into a router and firewall that you
manage with a single YAML file. You describe the state you want (networking,
routing, firewall, services) and `routier` renders and applies it. Every apply
comes with a safety net: the previous state is snapshotted and a rollback
watchdog is armed, so even a bad firewall rule cannot lock you out of a remote
box. A web UI and an HTTP API sit on top of the same engine, and everything
they can do has a CLI equivalent.

## Getting started

Turn a fresh Alpine Linux box into a Routier appliance:

1. **Install the package** from the signed Routier repository:

   ```bash
   setup-apkrepos -c -1   # enable Alpine's community repo (Routier deps live there)
   wget -qO /etc/apk/keys/routier.rsa.pub %%REPO_URL%%/routier.rsa.pub
   apk add --repository %%REPO_URL%%/nightly routier
   ```

   The post-install step backs up the host's network configuration to
   `/var/lib/routier/host-backup/` and only takes over networking if it can
   keep the box online (an existing config, or a DHCP fallback synthesized
   from the current default route).

2. **Start the services**:

   ```bash
   rc-service routier start
   rc-service routier-ui start
   ```

3. **Open the web UI** on port 8080 and sign in with the default credentials
   `routier` / `admin`. A fresh install walks you through a short onboarding
   wizard: set the admin password, hostname, and interfaces (a
   guided uplink DHCP setup plus an optional static LAN), then review and
   apply.

4. **Iterate declaratively.** From here on, you edit sections in the UI or in
   `/etc/routier/config.yml` and apply. Each apply snapshots the previous state
   and arms the rollback watchdog: confirm the apply once you can still reach
   the box, or let the watchdog restore the previous state for you.

See [installation.md](installation.md) for the takeover details and upgrades,
and [configuration.md](configuration.md) for the config model.

## How it fits together

- **Config**. One declarative YAML document (`/etc/routier/config.yml`, or a
  directory of YAML files merged together). See
  [configuration.md](configuration.md).
- **Render**. The config is turned into concrete artifacts: interface setup,
  FRR, nftables, wg-quick, service configs.
- **Apply**. Artifacts are applied atomically with a snapshot and an armed
  rollback (`routier apply` / `POST /api/config/apply`).
- **Surfaces**. The web UI, the HTTP API, and the `routier` CLI are three views
  of the same engine and the same config.

## AI usage

This project welcomes AI-assisted contributions. I build it with tools like
Claude Code, and you are free to use Claude, Codex, or your hands. Whatever you
use, own the result: know what the code does, test it, and be ready to explain
it. See [code-style.md](code-style.md) for the coding style and other rules,
and how I review the backend and the frontend differently.

