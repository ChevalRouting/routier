#!/bin/sh
set -eu

usage='usage: REPO_TARGET=<directory> repo.sh <command>
  add <channel> <arch> <version> <release>   publish dist/<arch> packages, then prune the channel
  remove <channel> <arch> <version>-r<release>
  promote <arch> <version>-r<release>        copy a nightly build into stable
  prune <channel> <arch>                     drop packages older than KEEP_DAYS
  reindex <channel> <arch>
  list
  iso <arch>                                 publish the newest dist/iso image, keep ISO_KEEP per arch
  counter next <arch> <version>              print the next release number of a version
  counter set <arch> <version> <release>
  site                                       mirror website/build into the directory, keeping repo/
  page                                       rewrite the index.html of every directory
  receive                                    unpack a tar of <arch>/*.apk and iso/*.iso from stdin into dist/'

die() {
	echo "$*" >&2
	exit 1
}

receive() {
	inbox=$(mktemp "$root/dist/.receive.XXXXXX")
	trap 'rm -f "$inbox"' EXIT
	cat > "$inbox"
	tar -tf "$inbox" | while read -r name; do
		echo "$name" | grep -Eq '^((x86_64|aarch64)/[A-Za-z0-9._-]+\.apk|iso/[A-Za-z0-9._-]+\.iso)$' || die "refusing $name"
	done
	tar -xf "$inbox" -C "$root/dist" --no-same-owner --no-same-permissions
	tar -tf "$inbox" | sed 's/^/received /'
}

[ $# -ge 1 ] || die "$usage"
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$root/dist/iso" "$root/dist/x86_64" "$root/dist/aarch64"

if [ "$1" = receive ]; then
	[ $# -eq 1 ] || die "$usage"
	receive
	exit 0
fi

TARGET="${REPO_TARGET:?$usage}"
mkdir -p "$TARGET"
TARGET=$(CDPATH= cd -- "$TARGET" && pwd)
SIGN_KEY="${SIGN_KEY:-$root/routier.rsa}"
IMAGE="${REPO_IMAGE:-routier-repo:latest}"
CONTAINER="${CONTAINER:-$(command -v podman >/dev/null 2>&1 && echo podman || echo docker)}"
command_args="$*"

set -- -e TARGET=/target -e KEEP_DAYS -e ISO_KEEP -e REPO_URL -e SITE_URL \
	-v "$TARGET:/target" \
	-v "$root/scripts/repo:/repo:ro" \
	-v "$root/assets/browse-routier.html:/browse.html:ro" \
	-v "$root/dist/iso:/iso:ro"

case "$command_args" in
	site)
		[ -d "$root/website/build" ] && set -- "$@" -v "$root/website/build:/site:ro"
		;;
	add\ *)
		pkgs="$root/dist/$(echo "$command_args" | cut -d ' ' -f 3)"
		[ -d "$pkgs" ] && set -- "$@" -v "$pkgs:/pkgs:ro"
		;;
esac

if [ -f "$SIGN_KEY" ]; then
	set -- "$@" -v "$SIGN_KEY:/key/routier.rsa:ro" -v "$SIGN_KEY.pub:/key/routier.rsa.pub:ro"
fi

export KEEP_DAYS="${KEEP_DAYS:-}" ISO_KEEP="${ISO_KEEP:-5}"
export REPO_URL="${PUBLIC_REPO_URL:-https://routier.chevalrouting.org/repo}" SITE_URL="${SITE_URL:-https://routier.chevalrouting.org}"
[ -n "${REPO_IMAGE:-}" ] || "$CONTAINER" build -q -t "$IMAGE" \
	--build-arg ALPINE_IMAGE="${ALPINE_IMAGE:-docker.io/library/alpine:3.23}" "$root/scripts/repo" >/dev/null

lock=
command -v flock >/dev/null 2>&1 && lock="flock $root/dist/.repo.lock"
exec $lock "$CONTAINER" run --rm "$@" "$IMAGE" $command_args
