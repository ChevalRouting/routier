# Building and testing packages

Routier ships as a signed Alpine `apk`. Everything builds inside a podman
container, so the host only needs `podman`, `make` and `git`; any Linux
distro works, not just Alpine.

## Build the packages

```bash
git clone %%GITHUB_URL%% && cd routier
make apk VERSION=0.5.4
```

What happens:

1. `make builder` builds the Alpine builder image from the repo `Dockerfile`
   (done automatically on first run).
2. A signing key pair `routier.rsa` / `routier.rsa.pub` is generated at the
   repo root if none exists. Keep the private key out of git (it is ignored)
   and reuse the same key for every build you publish; after a key change,
   clients refuse the repository.
3. `scripts/mkpkg.sh` runs `abuild` in the container: it builds the Go
   binaries with the embedded web UI (`EMBED=1`), packages `routier` and
   `routier-openrc`, and signs them.

Signed packages land in `dist/<arch>/`.

Useful variants:

```bash
make apk VERSION=0.5.4 RELEASE=1     # bump the -rN package release
make apk VERSION=0.5.4 LOCAL_ANYK=1  # build against a local ../anyk checkout
make apk_debug VERSION=0.5.4         # interactive shell in the builder image
```

## Assemble and serve a repository

```bash
make apk-index VERSION=0.5.4   # builds + signs APKINDEX.tar.gz in dist/<arch>/
make www VERSION=0.5.4         # copies apks + index + public key to WWW_DIR
```

`WWW_DIR` (default `/var/lib/Routier`) can be served by any static HTTP
server. Clients consume it with:

```bash
wget -qO /etc/apk/keys/routier.rsa.pub %%REPO_URL%%/routier.rsa.pub
apk add --repository %%REPO_URL%% routier
```

## Test a package

- `make test` runs `go vet` and the full test suite. The integration tests in
  `tests/` use a mocked service runner, so they exercise render/apply/rollback
  logic without touching the host (see [testing.md](testing.md)).
- `make iso VERSION=0.5.4` builds a bootable ISO that preinstalls the freshly
  built packages (clones Alpine aports on first run).
- `make run-iso` boots the ISO in QEMU; `make run-disk` boots the installed
  disk image. This is the quickest way to test install, takeover, firstboot
  and onboarding end to end.
- To test an upgrade on a live box, point it at your repository and run
  `apk upgrade routier`.

## Documentation site

```bash
make website-serve   # live-reload preview at http://localhost:3000
make website         # static build in website/build/
```

Host it anywhere static; for a custom domain, build with
`SITE_URL=%%SITE_URL%% make website`. Public URLs come from one place
(`website/src/remark/domains.ts`); override them with `SITE_URL` / `REPO_URL`.

### Versioning

The repo-root `docs/` is always the in-development version, edited in place. At
release time, snapshot it as a released version:

```bash
make website-version VERSION=0.5   # snapshot current docs as the 0.5 line
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
2. Builds and signs the packages with `make apk`, using the `ROUTIER_RSA`
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
git pull && make www VERSION=0.5.4
```
