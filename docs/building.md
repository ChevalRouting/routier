# Building and testing packages

Routier ships as a signed Alpine `apk`. Everything builds inside a podman
container, so the host only needs `podman`, `task` and `git`; any Linux
distro works, not just Alpine.

## Build the packages

```bash
git clone %%GITHUB_URL%% && cd routier
task apk VERSION=0.5.4
```

What happens:

1. `task builder` builds the Alpine builder image from the repo `Dockerfile`
   (done automatically on first run).
2. A signing key pair `routier.rsa` / `routier.rsa.pub` is generated at the
   repo root if none exists. Keep the private key out of git (it is ignored)
   and reuse the same key for every build you publish; after a key change,
   clients refuse the repository.
3. `scripts/mkpkg.sh` runs `abuild` in the container: it builds the Go
   binaries with the embedded web UI (`EMBED=1`), packages `routier` and
   `routier-openrc`, and signs them.

Signed packages land in `dist/<arch>/`.

## Offline builds

The apk and ISO builds normally reach the network for npm packages (`npm ci`)
and Alpine packages (the aports mirror). Prime both caches once while online:

```bash
task prefetch VERSION=0.5.4
```

This fills a gitignored cache under `dist/cache/`: `dist/cache/npm` (the npm
download cache) and `dist/cache/apkmirror/<arch>` (the full ISO package closure
with a routier-signed `APKINDEX.tar.gz`). `task prefetch` depends on `task apk`,
so routier's own packages exist first.

After that, both builds run without network access:

```bash
task apk VERSION=0.5.4              # npm ci --prefer-offline uses dist/cache/npm
task iso OFFLINE=1 VERSION=0.5.4   # mkimage uses the local mirror only
```

`OFFLINE=1` points the ISO build at `dist/cache/apkmirror` instead of the CDN
and fails if the mirror is missing. Without it, the ISO build uses the CDN as
before. Re-run `task prefetch` after changing npm or Go dependencies (a Go
dependency change also rebuilds the builder image, where Go modules are primed).
If an offline ISO build reports a missing Alpine package, add it to the fetch
list in `scripts/prefetch.sh` and re-run `task prefetch`.

## Build the MCP binary

Build the standalone Linux MCP server and place it beside the other
architecture-specific artifacts:

```bash
task mcp VERSION=0.5.4
```

The output is `dist/<arch>/routier-mcp`. Override `ARCH=aarch64` or
`ARCH=x86_64` to select the target
architecture; the MCP server is built without CGO and can be copied directly
to another Linux host.

The MCP binary is a controller-side tool. It is deliberately not included in
the Routier APK or appliance ISO and should not run on a managed router.

By default the build uses the pinned development-tool container. To use the
Go installation on the host and avoid Docker entirely:

```bash
task mcp NATIVE=true VERSION=0.5.4
```

Useful variants:

```bash
task apk VERSION=0.5.4 RELEASE=1     # bump the -rN package release
task apk VERSION=0.5.4 LOCAL_ANYK=1  # build against a local ../anyk checkout
task apk_debug VERSION=0.5.4         # interactive shell in the builder image
```

## Local repository

`task apk` leaves a signed repository in `dist/<arch>/`: the packages, a signed
`APKINDEX.tar.gz` and the public key. Serve `dist/` with any static HTTP
server to test an upgrade on a lab box, or lay it out like the public
repository with `task publish REPO_TARGET=/some/dir` (see
[Package repository](#package-repository)).

## Test a package

- `task test` runs `go vet` and the full test suite. The integration tests in
  `tests/` use a mocked service runner, so they exercise render/apply/rollback
  logic without touching the host (see [testing.md](testing.md)).
- `task iso VERSION=0.5.4` builds a bootable ISO that preinstalls the freshly
  built packages (clones Alpine aports on first run). `ARCH` selects the target
  CPU (`x86_64` or `aarch64`, default the build host) and `FLAVOR` selects the
  kernel: `lts` (Alpine long-term-support, bare metal), `virt` (VMs), or
  `stable` (mainline stable, from the community repo), default `lts`. The arch is
  encoded in the filename, so a mismatched arch will not boot the target
  hardware. `task iso-all` builds every arch/flavor combination.
- `task iso FLAVOR=installer` builds a light installer ISO instead of the
  run-from-RAM appliance. The live system boots a minimal RAM footprint (disk
  tools, DHCP on every interface, the `routier` binary and the web UI, no
  router daemons), and the full appliance closure rides along as an on-media
  apk cache. Install from the web UI or with `setup-routier` on the console
  (see [installation.md](installation.md#installation-media)). This is the path
  for machines that cannot hold the full stack in a tmpfs.
- `task cloud VERSION=0.5.4` builds a UEFI qcow2 cloud image under
  `dist/cloud`, provisioned at boot by cloud-init (see
  [installation.md](installation.md#cloud-images)). The image is 2G and grows
  with the instance disk; `SIZE` and `KFLAVOR` (default `virt`) override the
  defaults. It needs a privileged container for loop devices.
- `task run-iso` boots the ISO in QEMU; `task run-disk` boots the installed
  disk image. This is the quickest way to test install, takeover, firstboot
  and onboarding end to end.
- To test an upgrade on a live box, point it at a repository (a
  [local one](#local-repository) or a channel) and run `apk upgrade routier`.

## Documentation site

```bash
task website-serve   # live-reload preview at http://localhost:3000
task website         # static build in website/build/
```

Host it anywhere static; for a custom domain, build with
`SITE_URL=%%SITE_URL%% task website`. Public URLs come from one place
(`website/src/remark/domains.ts`); override them with `SITE_URL` / `REPO_URL`.

### Versioning

The repo-root `docs/` is always the in-development version, edited in place. At
release time, snapshot it as a released version:

```bash
task website-version VERSION=0.5   # snapshot current docs as the 0.5 line
```

This writes `website/versioned_docs/version-0.5/` and updates
`website/versions.json`; commit both. Use the `MAJOR.MINOR` line, not the patch
version, so patch releases reuse the same docs.

Once at least one version exists, the in-development `docs/` is hidden from the
built site: `/docs` serves the newest released version and the navbar dropdown
switches between the released lines.

## Database generation

The host development prerequisites are Docker, Task, and Git. Go, sqlc, swag,
oapi-codegen, OpenAPI Generator, Java, and Node-based generation tools run from
the pinned `tools` image defined in `dev/docker-compose.yml`.

Build or refresh that image explicitly with:

```bash
task tools
```

The standard `task build`, `task test`, `task lint`, `task sqlc`, and
`task docs` targets use this image automatically. Named volumes retain Go build,
Go module, and npm download caches, while generated files remain owned by the
host user.

Pass `NATIVE=true` to any of these targets to run the corresponding Go, Node,
sqlc, swag, oapi-codegen, Java, and lint commands directly on the host instead.
The required commands must already be installed and available on `PATH`.

Database schema migrations and sqlc queries live under `pkg/db`. Regenerate the
committed Go package after changing either source:

```bash
task sqlc
```

`task generate-check` verifies that the committed generated package matches its
schema and query sources. Runtime database upgrades use the same embedded
migrations; sqlc is not required on an appliance.

## Package repository

The public repository is a static tree served by Caddy:

```
%%REPO_URL%%/
  routier.rsa.pub            public signing key
  nightly/<arch>/            a build of every push to main, kept 30 days
  stable/<arch>/             tagged releases (v*), kept
  iso/                       installation media, routier-latest-<arch>.iso
```

Clients add a channel to `/etc/apk/repositories`, so `apk upgrade` and the
System › General › Updates page follow it:

```bash
wget -qO /etc/apk/keys/routier.rsa.pub %%REPO_URL%%/routier.rsa.pub
echo %%REPO_URL%%/nightly >> /etc/apk/repositories
apk update && apk add routier
```

ISO and cloud builds write the same line into the installed system:
`task iso CHANNEL=stable` follows the stable channel, and `PUBLIC_REPO_URL`
points the image at another repository. Offline ISOs (`OFFLINE=1`) list no
repository at all.

### Publish

```bash
task publish REPO_TARGET=routier@repo-host:/srv/routier/repo VERSION=0.5.4 RELEASE=1
task publish REPO_TARGET=routier@repo-host:/srv/routier/repo VERSION=0.6.0 CHANNEL=stable
```

`scripts/repo-publish.sh` runs in an Alpine container on the build machine. It
pulls the packages of the channel, adds the new `routier` and `routier-openrc`
apks, prunes nightly packages older than `KEEP_DAYS` (30), rebuilds and signs
`APKINDEX.tar.gz`, and pushes the result: packages first, then the index, then
the pruning, so clients never see an index that names a missing package. The
private key never leaves the build machine; the host only stores files.
`REPO_TARGET` may also be a local directory, which is the quickest way to test
a change to the script.

SSH goes through `SSH_KEY` when set (CI), otherwise through the ssh agent
(OrbStack and Docker Desktop forward it on macOS). Host keys come from
`KNOWN_HOSTS`, default `~/.ssh/known_hosts`.

### Continuous packaging (GitHub Actions)

`.github/workflows/nightly.yml` runs on every push to `main`, on `v*` tags and
on manual `workflow_dispatch`. On the self-hosted runner it:

1. computes the channel and version: `<last tag>_git<commit count>` on the
   nightly channel (for example `0.5.4_git128`, monotonic and valid for an apk
   `pkgver`), the tag without its `v` on the stable channel,
2. builds and signs the packages with `task apk`,
3. publishes them with `scripts/repo-publish.sh`,
4. builds this documentation site and deploys it to the same host.

The workflow needs these repository secrets and variables:

- secrets: `ROUTIER_RSA` (PEM private signing key), `ROUTIER_RSA_PUB`,
  `ROUTIER_STATIC_SSH_KEY`, `ROUTIER_STATIC_KNOWN_HOSTS`
- variables: `ROUTIER_STATIC_USER`, `ROUTIER_STATIC_HOST`, and optionally
  `ROUTIER_REPO_PATH` (default `/srv/routier/repo`) and `ROUTIER_STATIC_PATH`
  (default `/srv/routier/site`)

Use the same signing key as your manual builds so CI and hand-published
packages are interchangeable for clients.

### Host

One small VM serves the site and the repository. On Debian or Ubuntu:

1. point DNS for the site (`routier.cloud`) and the repository
   (`repo.routier.cloud`) at the VM, and open ports 80 and 443;
2. install Caddy from its official apt repository, plus `rsync`;
3. create the deploy user and the web roots it owns:

   ```bash
   sudo adduser --disabled-password --gecos "" routier
   sudo install -d -o routier -g routier /srv/routier/site /srv/routier/repo
   ```

4. add the CI and maintainer SSH keys to `~routier/.ssh/authorized_keys`;
5. install the Caddy configuration: `/etc/caddy/Caddyfile` serves
   `/srv/routier/site` on the site host, redirects the old `/repo/` paths and
   serves `/srv/routier/repo` with a browsable index on the repository host,
   with certificates from Let's Encrypt. `ROUTIER_SITE_HOST` and
   `ROUTIER_REPO_HOST` in Caddy's environment override the host names.

The first `task publish` creates the channel directories and uploads the
public key.

## Task runner

Build commands are defined in `Taskfile.yml` (Task v3). Install Task with
`brew install go-task` on macOS, or follow the [Task installation guide](https://taskfile.dev/docs/installation).
Run `task --list` to see the available commands; running `task` builds all binaries.
Options retain the `NAME=value` syntax, for example `task build NATIVE=true EMBED=1`.
Individual binaries can be built with `task build-routier` or `task build-routier-ui`.
Container builds remain the default; `NATIVE=true` uses tools installed on the host.
Packaging allocates a terminal only when run interactively; set `PODMAN_TTY=`
to disable it explicitly or `PODMAN_TTY=-it` to force it.

## Deploy directly to a router

```bash
task deploy HOST=root@router VERSION=0.5.4 RELEASE=1
```

`HOST` accepts an SSH config alias or `user@host`; SSH config controls the port,
identity file, and jump host. Deployment detects the remote Alpine architecture
and reuses `routier` and `routier-openrc` APKs in `dist/<arch>/` matching the exact
`VERSION` and `RELEASE`. It builds the packages only if either APK is missing,
then uploads both with the public signing key and installs them with
`apk add --upgrade`.
Non-root SSH users need `doas` or `sudo`. The signing public key is installed as
`/etc/apk/keys/routier.rsa.pub`; the private key stays on the build machine.

Use a new `VERSION` or increment `RELEASE` for each deployment. An already
installed version is rejected to avoid silently skipping a rebuilt package.
Existing package hooks handle the UI restart; deployment does not explicitly
reapply network configuration or reboot the host. Uploaded files are removed
on success and retained with a diagnostic path on failure.

Run `task test-deploy` to check the deployment scripts without contacting a host.
