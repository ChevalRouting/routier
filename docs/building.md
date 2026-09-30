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

The output is `dist/<arch>/routier-mcp`. `task mcp-dist` is an equivalent
explicit target. Override `ARCH=aarch64` or `ARCH=x86_64` to select the target
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

## Assemble and serve a repository

```bash
task apk-index VERSION=0.5.4   # builds + signs APKINDEX.tar.gz in dist/<arch>/
task www VERSION=0.5.4         # copies apks + index + public key to WWW_DIR
```

`WWW_DIR` (default `/var/lib/Routier`) can be served by any static HTTP
server. Clients consume it with:

```bash
wget -qO /etc/apk/keys/routier.rsa.pub %%REPO_URL%%/routier.rsa.pub
apk add --repository %%REPO_URL%% routier
```

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
- `task run-iso` boots the ISO in QEMU; `task run-disk` boots the installed
  disk image. This is the quickest way to test install, takeover, firstboot
  and onboarding end to end.
- To test an upgrade on a live box, point it at your repository and run
  `apk upgrade routier`.

## Documentation site

```bash
task website-serve   # live-reload preview at http://localhost:3000
task website         # static build in website/build/
```

Host it anywhere static; for a custom domain, build with
`SITE_URL=%%SITE_URL%% task website`. Public URLs come from one place
(`website/src/remark/domains.ts`); override them with `SITE_URL` / `REPO_URL`.

## Database generation

The host development prerequisites are Docker, Make, and Git. Go, sqlc, swag,
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

## Continuous packaging (GitHub Actions)

`.github/workflows/nightly.yml` runs on every push to `main` (and on manual
`workflow_dispatch`). On the self-hosted runner it:

1. Computes a version `<last tag>_git<commit count>` (for example
   `0.5.4_git128`), a monotonic value valid for an apk `pkgver`.
2. Builds and signs the packages with `task apk`, using the `ROUTIER_RSA`
   secret (PEM private key) and the `ROUTIER_RSA_PUB` variable (public key).
3. rsyncs the new apks over SSH to the repository host under
   `nightly/<arch>/`, publishes the public key at the repository root, then
   regenerates and signs `APKINDEX.tar.gz` on the host. Packages older than
   30 days are pruned.
4. Builds this documentation site and deploys it to the same host.

Consume the nightly channel like any apk repository:

```bash
wget -qO /etc/apk/keys/routier.rsa.pub %%REPO_URL%%/routier.rsa.pub
apk add --repository %%REPO_URL%%/nightly routier
```

The workflow needs these repository secrets and variables:

- secrets: `ROUTIER_RSA` (PEM private signing key), `ROUTIER_STATIC_SSH_KEY`,
  `ROUTIER_STATIC_KNOWN_HOSTS`
- variables: `ROUTIER_RSA_PUB`, `ROUTIER_STATIC_USER`, `ROUTIER_STATIC_HOST`,
  `ROUTIER_STATIC_PATH`, `ROUTIER_REPO_PATH`

Use the same signing key as your manual builds so the CI and self-hosted
repositories are interchangeable for clients.

## Manual release flow

The manual builder loop still works unchanged for a self-hosted repository;
only the signing key must match:

```bash
git pull && task www VERSION=0.5.4
```

### Task runner

Build commands are defined in `Taskfile.yml` (Task v3). Install Task with
`brew install go-task` on macOS, or follow the [Task installation guide](https://taskfile.dev/docs/installation).
Run `task --list` to see the available commands; running `task` builds all binaries.
Options retain the `NAME=value` syntax, for example `task build NATIVE=true EMBED=1`.
Individual binaries can be built with `task build-routier` or `task build-routier-ui`.
Container builds remain the default; `NATIVE=true` uses tools installed on the host.
Packaging allocates a terminal only when run interactively; set `PODMAN_TTY=`
to disable it explicitly or `PODMAN_TTY=-it` to force it.

### Deploy directly to a host

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
