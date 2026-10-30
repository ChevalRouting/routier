# Help test Routier

Routier is young and changes quickly. The most useful thing anyone can do
right now is run it on networks and hardware we do not have, and tell us what
broke, what confused you and what you missed. You do not need to read the code
or write Go to help.

## Before you start

- Start in a VM or a lab network: [Try Routier](quickstart.md) gets you there
  in about ten minutes. Move to a real edge only once you are comfortable with
  rollback and with getting back in from a console.
- Nightly builds come straight from the main branch. They are tested, but a
  regression can slip through; keep console or out-of-band access to any box
  you upgrade.
- Every apply is reversible: an unconfirmed apply rolls back after 60 seconds,
  and the apply history and snapshots keep earlier states. A package removal
  restores the host's original network configuration (see
  [Removal](installation.md#removal)).

## Where feedback helps most

| Area | What to look at |
|------|-----------------|
| Installation | the ISO on real hardware: network cards detected, UEFI and legacy boot, disk selection, the first boot after install |
| Onboarding | whether the wizard picks the right uplink, and whether each step is clear without reading the docs |
| Upgrades | moving from one nightly to the next through System › General › Updates, with your configuration intact |
| VPN | WireGuard tunnels to phones, laptops and other routers |
| Routing | BGP and OSPF sessions with your existing routers, route policies, BFD |
| High availability | two routers sharing a virtual address with VRRP and conntrackd |
| Switches | MikroTik RouterOS 7 switches managed from Routier ([Switches](switches.md)) |
| Monitoring | live traffic, logs, and sFlow export to a collector such as akvorado ([Monitoring](monitoring.md)) |
| Mobile | the web UI on a phone, including the installed home screen app |

Small observations count: a label that did not say what it does, a page that
needed a reload, an error message that did not help.

## Report a problem

Open an issue on [GitHub](%%GITHUB_URL%%/issues/new/choose) and pick the bug
report form. The form asks for:

- the Routier version (`routier version`, or System › General › Updates),
- how it was installed (ISO, cloud image or package) and on what hardware or
  hypervisor,
- what you did, what you expected and what happened instead,
- the relevant part of `/etc/routier/config.yml`, with keys, passwords and
  public addresses removed.

### When an apply fails

A failed apply opens a dialog with the failing step and the files Routier
rendered. Routier keeps the same information as a bundle under
`/var/lib/routier/failures/`. Attach it to the issue; the export replaces
secrets with placeholders. With an API key from System › General › API Keys:

```bash
curl -k -H "Authorization: Bearer $KEY" https://router:8080/api/failures
curl -k -H "Authorization: Bearer $KEY" -o failure.tar.gz \
  https://router:8080/api/failures/<id>/export
```

### Logs

Monitoring › System › Logs streams the system log and the kernel log. From a
shell, `/var/log/messages` holds the same lines, and `routier history` lists
recent applies with their result.

## Ideas and questions

Missing features, confusing workflows and "how would I do X" questions are
welcome too: open an issue with the feedback form. Describing the network you
are trying to build is often more useful than a proposed solution.

## Security issues

Do not open a public issue for a vulnerability. Report it privately through
the repository's Security tab on GitHub, with the version and the steps to
reproduce it.
