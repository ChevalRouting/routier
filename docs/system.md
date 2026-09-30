# System (hostname, DNS, SSH, sysctl, users, services)

These sections cover host-level configuration. UI: **System** group.

## Hostname

```yaml
hostname: gw1
```

## DNS

```yaml
dns:
  nameservers: ["1.1.1.1", "2606:4700:4700::1111"]
  search: ["example.com"]
```

These keys configure `/etc/resolv.conf` only - what the router itself resolves
against. They are unrelated to the DNS server Routier can run for the network,
which lives under `dns.server` and is documented in [dns.md](dns.md).

## SSH

```yaml
ssh:
  port: 22
  permit_root_login: "no"
  password_auth: "no"
  pubkey_auth: "yes"
  allow_users: [admin]
```

Fields: `port`, `permit_root_login`, `password_auth`, `pubkey_auth`,
`allow_tcp_forwarding`, `x11_forwarding`, `max_auth_tries`,
`login_grace_time`, `client_alive_interval`, `client_alive_count_max`,
`allow_users`, `allow_groups`, `banner`.

## sysctl

```yaml
sysctl:
  net.ipv4.ip_forward: "1"
  net.ipv6.conf.all.forwarding: "1"
```

A map of kernel sysctl keys to values.

## Users

```yaml
users:
  admin:
    uid: 1000
    shell: /bin/bash
    groups: [wheel]
    ssh_keys: ["ssh-ed25519 AAAA..."]
    password_hash: "$6$..."
```

Fields: `uid`, `shell`, `home`, `groups`, `ssh_keys`, `system`,
`password_hash`. Note that UI accounts (web login) are managed separately from
these system users.

## Services

```yaml
services:
  chrony:
    enable: true
    after: network
    configs:
      - { src: chrony.conf.tmpl, ... }
```

Each service has `enable`, an optional `after` ordering hint, and `configs`
(config files rendered from templates in the user template dir).

## Other host settings

- `boot_modules` - kernel modules to load at boot.
- `gai` - `gai.conf` address-selection policy (IPv4/IPv6 preference).
- `logging` - logging configuration.
