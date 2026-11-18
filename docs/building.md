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

The public repository is a tree of files in a directory, served by any static
HTTP server; `apk` checks the index signature:

```
%%REPO_URL%%/
  index.html                       how to add the repository; every directory has a listing page
  routier.rsa.pub                  public signing key
  nightly/<arch>/                  builds of main, kept 30 days
  stable/<arch>/                   tagged releases (v*), kept
  iso/                             installation media, routier-latest-<arch>.iso and
                                   routier-latest-installer-virt-<arch>.iso
  cloud/                           cloud-init qcow2 images, routier-latest-cloud-<arch>.qcow2
  state/releases/<arch>/<version>  last release number published for a version
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

The scripts write into the directory named by `REPO_TARGET`:

```bash
export REPO_TARGET=/srv/public/repo
export SIGN_KEY=/path/to/routier.rsa
```

`SIGN_KEY` defaults to `routier.rsa` at the root of the work tree, next to
`routier.rsa.pub`. The private key never leaves the build machine. Set
`ALPINE_IMAGE` and `NODE_IMAGE` to use mirrored images.

### Release

```bash
sh scripts/release.sh
SITE_TARGET=/srv/public ISO_FLAVORS="installer lts" sh scripts/release.sh
REPO_HOST=publisher@repo.example.net sh scripts/release.sh
```

`scripts/release.sh` builds the checked out commit, which must have no local
changes:

1. it picks the channel and version: the tag without its `v` on the stable
   channel when the commit carries a `v*` tag, otherwise
   `<last tag>_git<commit count>` on the nightly channel (for example
   `0.5.4_git128`, monotonic and valid for an apk `pkgver`);
2. it reads the next release number of that version from
   `state/releases/<arch>/<version>`, 0 for a new version, so builders of
   different architectures number the same commit alike;
3. it builds and signs the packages with `task apk` and publishes them;
4. it builds each ISO flavor of `ISO_FLAVORS` (default `installer installer-virt`) and
   publishes it;
5. it builds the cloud-init qcow2 image with `task cloud` and publishes it,
   unless `CLOUD_IMAGE=0`;
6. it records the release number, so a rebuild of the same commit gets the
   next one;
7. with `SITE_TARGET`, it builds this documentation site and mirrors it there,
   leaving the `repo/` directory alone so the repository can live inside it.

With `REPO_HOST`, the release builds locally and publishes on another machine:
it streams the packages and the ISO over SSH (`REPO_SSH`, default `ssh`) to
`repo.sh receive` there, then runs the repository commands there. The SSH
login on that host must run `scripts/repo.sh` with the requested arguments,
for example through a forced command, so only one machine writes the
repository and the release counters stay in one place.

`VERSION`, `CHANNEL`, `RELEASE` and `ARCH` override the computed values.
`task publish REPO_TARGET=...` runs the same script.

### Repository lifecycle

`scripts/repo.sh`, or `task repo REPO_TARGET=... -- <command>`, manages what
the release script published. It builds the `routier-repo` helper image from
`scripts/repo/Dockerfile` (Alpine with Python, abuild and openssl, based on
`ALPINE_IMAGE`) and runs `scripts/repo/repo.py` in it with `REPO_TARGET`
mounted. Set `REPO_IMAGE` to use an image built elsewhere instead. Commands
take a lock on `dist/.repo.lock` when `flock` is available.

| Command | Effect |
| --- | --- |
| `add <channel> <arch> <version> <release>` | publish `dist/<arch>` packages, then prune the channel |
| `remove <channel> <arch> <version>-r<release>` | drop a build from a channel |
| `promote <arch> <version>-r<release>` | copy a nightly build into stable |
| `prune <channel> <arch>` | drop packages older than `KEEP_DAYS` (30 for nightly, never for stable) |
| `reindex <channel> <arch>` | rebuild and sign the index |
| `list` | list packages and images |
| `iso <arch> [flavor]` | publish the newest `dist/iso` image of a flavor (default `installer`) as `routier-latest-<arch>.iso`, or `routier-latest-<flavor>-<arch>.iso` for other flavors, keeping `ISO_KEEP` (5) per flavor |
| `cloud <arch>` | publish the newest `dist/cloud` image as `routier-latest-cloud-<arch>.qcow2`, keeping `ISO_KEEP` (5) |
| `counter next <arch> <version>`, `counter set <arch> <version> <release>` | read or set the release counter |
| `site` | mirror `website/build` into the directory, deleting stale files but keeping `repo/` |
| `page` | rewrite the listing page of every directory, styled by `assets/browse-routier.html`; also done after every channel change and image upload |
| `receive` | unpack a tar of `<arch>/*.apk`, `iso/*.iso` and `cloud/*.qcow2` from stdin into `dist/`, for `REPO_HOST` releases |

Every change to a channel copies its packages to a staging directory, rebuilds
and signs `APKINDEX.tar.gz`, checks that the index describes every package and
carries the signature, then publishes new packages first, the index next and
deletes pruned packages last, so clients never see an index that names a
missing package. Each file is written next to its destination and renamed into
place.

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
