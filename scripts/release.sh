#!/bin/sh
set -eu

usage='usage: REPO_TARGET=<directory> SIGN_KEY=<path> release.sh
  REPO_HOST=<user@host> publishes on that host over SSH instead of REPO_TARGET (REPO_SSH, default ssh)
  VERSION, CHANNEL and RELEASE default to the checked out commit and the next release number
  ISO_FLAVORS lists the ISO flavors to build and publish (default: installer, empty for none)
  SITE_TARGET=<directory> also builds the documentation site and publishes it there'

die() {
	echo "$*" >&2
	exit 1
}

repo() {
	if [ -n "${REPO_HOST:-}" ]; then
		$REPO_SSH "$REPO_HOST" "$*"
	else
		sh scripts/repo.sh "$@"
	fi
}

send() {
	[ -z "${REPO_HOST:-}" ] || tar -C dist -cf - "$@" | $REPO_SSH "$REPO_HOST" receive
}

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

[ -n "${REPO_HOST:-}" ] || : "${REPO_TARGET:?$usage}"
REPO_SSH="${REPO_SSH:-ssh}"
SIGN_KEY="${SIGN_KEY:-$root/routier.rsa}"
ARCH="${ARCH:-$(uname -m | sed s/arm64/aarch64/)}"
ISO_FLAVORS="${ISO_FLAVORS-installer}"
TASK="${TASK:-task}"
NODE_IMAGE="${NODE_IMAGE:-docker.io/library/node:24-bookworm-slim}"
CONTAINER="${CONTAINER:-$(command -v podman >/dev/null 2>&1 && echo podman || echo docker)}"

[ -f "$SIGN_KEY" ] && [ -f "$SIGN_KEY.pub" ] || die "signing key pair missing at $SIGN_KEY"
[ -z "$(git status --porcelain --untracked-files=no)" ] || die "the work tree has local changes, commit or stash them first"

tag=$(git describe --tags --exact-match --match 'v*' 2>/dev/null || true)
if [ -n "$tag" ]; then
	CHANNEL="${CHANNEL:-stable}"
	VERSION="${VERSION:-${tag#v}}"
else
	base=$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || echo v0.0.0)
	CHANNEL="${CHANNEL:-nightly}"
	VERSION="${VERSION:-${base#v}_git$(git rev-list --count HEAD)}"
fi

RELEASE="${RELEASE:-$(repo counter next "$ARCH" "$VERSION")}"
case "$RELEASE" in
	''|*[!0-9]*) die "cannot read the release counter of $VERSION: $RELEASE" ;;
esac

if [ "$(cd "$(dirname "$SIGN_KEY")" && pwd)/$(basename "$SIGN_KEY")" != "$root/routier.rsa" ]; then
	if [ -f routier.rsa ] && ! cmp -s "$SIGN_KEY" routier.rsa; then
		die "routier.rsa in the work tree is not the signing key at $SIGN_KEY"
	fi

	install -m 600 "$SIGN_KEY" routier.rsa
	install -m 644 "$SIGN_KEY.pub" routier.rsa.pub
fi

echo "building routier $VERSION-r$RELEASE ($(git rev-parse --short HEAD)) for $CHANNEL/$ARCH"
"$TASK" apk VERSION="$VERSION" RELEASE="$RELEASE" ARCH="$ARCH" PODMAN_TTY=
send "$ARCH/routier-$VERSION-r$RELEASE.apk" "$ARCH/routier-openrc-$VERSION-r$RELEASE.apk"
repo add "$CHANNEL" "$ARCH" "$VERSION" "$RELEASE"

for flavor in $ISO_FLAVORS; do
	"$TASK" iso VERSION="$VERSION" RELEASE="$RELEASE" ARCH="$ARCH" FLAVOR="$flavor" CHANNEL="$CHANNEL" PODMAN_TTY=
	send "iso/$(basename "$(ls -t dist/iso/*-"$ARCH".iso | head -1)")"
	repo iso "$ARCH"
done

repo counter set "$ARCH" "$VERSION" "$RELEASE"

if [ -n "${SITE_TARGET:-}" ]; then
	"$CONTAINER" run --rm \
		-v "$root/website:/src" \
		-v "$root/docs:/docs:ro" \
		-w /src \
		-e HOME=/tmp \
		-e npm_config_cache=/tmp/.npm-cache \
		-e SITE_URL="${SITE_URL:-https://routier.chevalrouting.org}" \
		-e REPO_URL="${PUBLIC_REPO_URL:-https://routier.chevalrouting.org/repo}" \
		-e SITE_COMMIT="$(git rev-parse --short HEAD)" \
		-e SITE_VERSION="$VERSION" \
		"$NODE_IMAGE" \
		sh -c 'npm ci --no-audit --no-fund && sh scripts/version-dev.sh && npm run build'
	REPO_TARGET="$SITE_TARGET" sh scripts/repo.sh site
fi

echo "released routier $VERSION-r$RELEASE to $CHANNEL/$ARCH"
