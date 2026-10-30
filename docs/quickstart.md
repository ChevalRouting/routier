# Try Routier

The quickest way to see Routier is a virtual machine booted from the
installation ISO. It takes about ten minutes and leaves your own network
untouched.

## What you need

- a hypervisor: QEMU/KVM, Proxmox, virt-manager, UTM, VirtualBox or VMware,
- a VM with 2 vCPUs, 2 GB of RAM and a 10 GB disk,
- two network adapters: the first one is the uplink (NAT or bridged to your
  network), the second one is the local network Routier serves (an internal
  or host-only network),
- optionally a second VM on that internal network, to see a client get an
  address from Routier's DHCP server.

## 1. Download the ISO

Installation media live under [%%REPO_URL%%/iso/](%%REPO_URL%%/iso/):

| File | Use |
|------|-----|
| `routier-latest-x86_64.iso` | the newest appliance ISO for 64-bit PCs and most VMs |
| `routier-<version>-x86_64.iso` | a specific build |

The appliance ISO runs the whole stack from memory, so it can be tried
without installing anything. Arm boards and Apple silicon VMs use the
`aarch64` images when they are published.

## 2. Boot the VM

Attach the ISO and boot. On QEMU, with the uplink forwarded to your machine:

```bash
qemu-img create -f qcow2 routier.qcow2 10G
qemu-system-x86_64 -enable-kvm -m 2G -smp 2 \
  -cdrom routier-latest-x86_64.iso \
  -drive file=routier.qcow2,if=virtio \
  -netdev user,id=wan,hostfwd=tcp::8080-:8080,hostfwd=tcp::2222-:22 \
  -device virtio-net-pci,netdev=wan \
  -netdev socket,id=lan,listen=:4444 \
  -device virtio-net-pci,netdev=lan
```

Every interface comes up with DHCP and the console prints where the web UI
listens, with a one-time password:

```
  This system runs from the installation media.

  Install from a browser:
    https://192.0.2.10:8080
  Sign in as routier with the password mB5avGD5ML6xT3FH
```

With the QEMU command above, open `https://localhost:8080` instead. The
certificate is self-signed until you configure one, so the browser asks you to
accept it once.

## 3. Install, or keep it live

After signing in, the web UI offers to install to the VM disk: pick the disk,
set the hostname, timezone and admin password, confirm, and restart without
the ISO. On the appliance ISO you can also continue without installing; the
system then runs from memory and forgets everything on restart.

## 4. Onboarding

The first sign-in starts a short wizard:

1. change the admin password,
2. set the hostname,
3. review the interfaces: the uplink keeps DHCP, and the second adapter
   becomes a local network with a static address and a DHCP pool,
4. review the changes and apply.

Every apply arms a rollback timer. Once the page reloads, confirm the apply;
if the change had cut you off, Routier would restore the previous state after
60 seconds on its own.

## 5. Things to try

- **Every section.** Interfaces, routing, VPN, firewall, DHCP, DNS and high
  availability, each with the diff shown before applying.
- **A port forward or a firewall rule**, then the diff shown before applying.
- **Rollback.** Apply a change and do not confirm it. After 60 seconds the
  previous configuration comes back, along with an entry in the apply history.
- **WireGuard.** Add a tunnel and a peer under VPN, then connect a phone or a
  laptop with the generated configuration.
- **Monitoring.** Watch traffic, logs and the routing table update live, and
  turn on flow sampling in System › General › Monitoring to see the traffic
  diagram fill in.
- **The same config as a file.** Everything the UI does ends up in
  `/etc/routier/config.yml`. Over SSH (port 2222 with the QEMU command above),
  `routier plan` shows pending changes and `routier apply` applies them.

## 6. Stay up to date

The ISO follows the nightly channel, which receives a build for every change.
System › General › Updates lists and installs new versions; `apk update && apk
upgrade` does the same from a shell.

## Other ways in

- [Cloud images](installation.md#cloud-images) for providers that take a qcow2
  image and cloud-init.
- [The package](installation.md#package-install) on an existing Alpine Linux
  box.

When something looks wrong, or just surprising, see
[Help test Routier](help-test.md) for what is most useful to report.
