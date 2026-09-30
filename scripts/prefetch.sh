#!/bin/sh

set -eu

export HERE=$(cd "$(dirname "$0")" && pwd)
export ARCH="$(apk --print-arch)"
export APORTS="${APORTS:-/aports}"
export REPO="${ROUTIER_REPO:-$HERE/../dist}"
export MIRROR="${ALPINE_MIRROR:-https://dl-cdn.alpinelinux.org/alpine/v3.23/main}"
export COMMUNITY="${ALPINE_COMMUNITY:-https://dl-cdn.alpinelinux.org/alpine/v3.23/community}"
export MIRROR_DIR="$HERE/../dist/cache/apkmirror"
export NPM_CACHE="$HERE/../dist/cache/npm"

[ -f "$REPO/$ARCH/APKINDEX.tar.gz" ] || { echo "no signed repo at $REPO/$ARCH, run task apk first" >&2; exit 1; }
[ -d "$APORTS/scripts" ] || { echo "set APORTS to an Alpine aports checkout" >&2; exit 1; }

mkdir -p /home/builder/.abuild
install -m 600 "${HERE}/../routier.rsa"     /home/builder/.abuild/routier.rsa
install -m 644 "${HERE}/../routier.rsa.pub" /home/builder/.abuild/routier.rsa.pub
printf 'PACKAGER_PRIVKEY="/home/builder/.abuild/routier.rsa"\n' > /home/builder/.abuild/abuild.conf
chown -R builder: /home/builder/.abuild

cp /home/builder/.abuild/*.pub /etc/apk/keys/ || true
cp "$REPO/$ARCH"/*.pub /etc/apk/keys/ 2>/dev/null || true

TMPWEB="$(mktemp -d)"
cp "$HERE/../web/package.json" "$HERE/../web/package-lock.json" "$TMPWEB/"
mkdir -p "$TMPWEB/vendor"
cp "$HERE/../web/vendor/cheval-ui.tgz" "$TMPWEB/vendor/"

mkdir -p "$NPM_CACHE"
( cd "$TMPWEB" && npm_config_cache="$NPM_CACHE" npm ci --ignore-scripts )
chmod -R a+rwX "$NPM_CACHE"
rm -rf "$TMPWEB"

cp "$HERE/mkimg.routier.sh"     "$APORTS/scripts/mkimg.routier.sh"
cp "$HERE/genapkovl-routier.sh" "$APORTS/scripts/genapkovl-routier.sh"

apks=""
set +u
. "$APORTS/scripts/mkimg.base.sh"
. "$APORTS/scripts/mkimg.routier.sh"
profile_routier
_apks_lts="$apks"
profile_routier_virt
_apks_virt="$apks"
profile_routier_stable
apks="$_apks_lts $_apks_virt $apks"
set -u

[ -n "$apks" ] || { echo "failed to resolve package list from aports profile" >&2; exit 1; }

ROOT="$(mktemp -d)"
mkdir -p "$ROOT/etc/apk/keys"
cp /etc/apk/keys/* "$ROOT/etc/apk/keys/" 2>/dev/null || true
if [ -d /usr/share/apk/keys/"$ARCH" ]; then
	cp /usr/share/apk/keys/"$ARCH"/* "$ROOT/etc/apk/keys/" 2>/dev/null || true
fi

apk --arch "$ARCH" --root "$ROOT" add --initdb
printf '%s\n%s\n%s\n' "$MIRROR" "$COMMUNITY" "$REPO" > "$ROOT/etc/apk/repositories"
apk update --root "$ROOT"

DEST="$MIRROR_DIR/$ARCH"
rm -rf "$DEST"
mkdir -p "$DEST"
apk fetch --root "$ROOT" --recursive --output "$DEST" $apks linux-firmware wireless-regdb mkinitfs

for extra in syslinux grub-ieee1275 grub-bios; do
	apk fetch --root "$ROOT" --recursive --output "$DEST" "$extra" >/dev/null 2>&1 || true
done

rm -f "$DEST"/routier-*.apk

( cd "$DEST" && apk index --rewrite-arch "$ARCH" -o APKINDEX.tar.gz *.apk )
chmod -R a+rwX "$MIRROR_DIR"
su builder -c ". /home/builder/.abuild/abuild.conf && cd '$DEST' && abuild-sign APKINDEX.tar.gz"

rm -rf "$ROOT"

echo "prefetch complete:"
echo "  npm cache:  $NPM_CACHE"
echo "  apk mirror: $DEST"
