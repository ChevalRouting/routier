#!/bin/sh

set -eu

export HERE=$(cd "$(dirname "$0")" && pwd)
export ARCH="$(apk --print-arch)"
export REPO="${ROUTIER_REPO:-$HERE/../dist}"
export APORTS="${APORTS:-/aports}"
export OUTDIR="${OUTDIR:-$HERE/../dist/iso}"
export MIRROR="${ALPINE_MIRROR:-https://dl-cdn.alpinelinux.org/alpine/v3.23/main}"
export COMMUNITY="${ALPINE_COMMUNITY:-https://dl-cdn.alpinelinux.org/alpine/v3.23/community}"

[ -f "$REPO/$ARCH/APKINDEX.tar.gz" ] || { echo "no signed repo at $REPO/$ARCH, run make apk first" >&2; exit 1; }
[ -d "$APORTS/scripts" ] || { echo "set APORTS to an Alpine aports checkout" >&2; exit 1; }

mkdir -p /home/builder/.abuild
install -m 600 "${HERE}/../routier.rsa"     /home/builder/.abuild/routier.rsa
install -m 644 "${HERE}/../routier.rsa.pub" /home/builder/.abuild/routier.rsa.pub
printf 'PACKAGER_PRIVKEY="/home/builder/.abuild/routier.rsa"\n' > /home/builder/.abuild/abuild.conf
chown -R builder: /home/builder/.abuild

cp /home/builder/.abuild/*.pub /etc/apk/keys/ || true
cp "$REPO/$ARCH"/*.pub /etc/apk/keys/ || true
cp "$HERE/mkimg.routier.sh"      "$APORTS/scripts/mkimg.routier.sh"
cp "$HERE/genapkovl-routier.sh" "$APORTS/scripts/genapkovl-routier.sh"

rm -rf "$OUTDIR"
mkdir -p "$OUTDIR"
chown builder: "$OUTDIR"

su builder -c ". /home/builder/.abuild/abuild.conf && cd '$APORTS/scripts' && $APORTS/scripts/mkimage.sh \
	--tag 'routier' \
	--outdir '$OUTDIR' \
	--arch '$ARCH' \
	--hostkeys \
	--repository '$MIRROR' \
	--repository '$COMMUNITY' \
	--repository '$REPO' \
	--profile routier"

chown -R 0:0 "$OUTDIR"

echo "ISO written to $OUTDIR"

ISO=$(ls -t "$OUTDIR"/*.iso 2>/dev/null | head -1)
if [ -n "$ISO" ]; then
	BOOTDIR="$HERE/../dist/boot"
	mkdir -p "$BOOTDIR"
	xorriso -osirrox on -indev "$ISO" \
		-extract /boot/vmlinuz-virt   "$BOOTDIR/vmlinuz-virt" \
		-extract /boot/initramfs-virt "$BOOTDIR/initramfs-virt" \
		2>/dev/null
	echo "kernel/initramfs extracted to $BOOTDIR"
fi
