#!/bin/sh

set -eu

export HERE=$(cd "$(dirname "$0")" && pwd)
export ARCH="$(apk --print-arch)"
export VERSION="${VERSION:-0.0.0}"
export RELEASE="${RELEASE:-0}"
export REPO="${ROUTIER_REPO:-$HERE/../dist}"
export APORTS="${APORTS:-/aports}"
export OUTDIR="${OUTDIR:-$HERE/../dist/iso}"
export MIRROR="${ALPINE_MIRROR:-https://dl-cdn.alpinelinux.org/alpine/v3.23/main}"
export COMMUNITY="${ALPINE_COMMUNITY:-https://dl-cdn.alpinelinux.org/alpine/v3.23/community}"
export OFFLINE="${OFFLINE:-0}"
export MIRROR_DIR="${MIRROR_DIR:-$HERE/../dist/cache/apkmirror}"
export FLAVOR="${FLAVOR:-lts}"
export ROUTIER_REPO_URL="${ROUTIER_REPO_URL-https://repo.routier.cloud}"
export ROUTIER_CHANNEL="${ROUTIER_CHANNEL:-nightly}"

case "$FLAVOR" in
	lts)       PROFILE="routier"; IMAGE="routier" ;;
	virt)      PROFILE="routier_virt"; IMAGE="routier-virt" ;;
	stable)    PROFILE="routier_stable"; IMAGE="routier-stable" ;;
	installer) PROFILE="routier_installer"; IMAGE="routier-installer" ;;
	*) echo "unknown FLAVOR=$FLAVOR (lts|virt|stable|installer)" >&2; exit 1 ;;
esac

[ -f "$REPO/$ARCH/APKINDEX.tar.gz" ] || { echo "no signed repo at $REPO/$ARCH, run task apk first" >&2; exit 1; }
[ -d "$APORTS/scripts" ] || { echo "set APORTS to an Alpine aports checkout" >&2; exit 1; }

if [ "$OFFLINE" = "1" ]; then
	[ -f "$MIRROR_DIR/$ARCH/APKINDEX.tar.gz" ] || { echo "OFFLINE=1 but no mirror at $MIRROR_DIR/$ARCH, run task prefetch first" >&2; exit 1; }
	REPOS="--repository '$MIRROR_DIR' --repository '$REPO'"
else
	REPOS="--repository '$MIRROR' --repository '$COMMUNITY' --repository '$REPO'"
fi

mkdir -p /home/builder/.abuild
install -m 600 "${HERE}/../routier.rsa"     /home/builder/.abuild/routier.rsa
install -m 644 "${HERE}/../routier.rsa.pub" /home/builder/.abuild/routier.rsa.pub
printf 'PACKAGER_PRIVKEY="/home/builder/.abuild/routier.rsa"\n' > /home/builder/.abuild/abuild.conf
chown -R builder: /home/builder/.abuild

cp /home/builder/.abuild/*.pub /etc/apk/keys/ || true
cp "$REPO/$ARCH"/*.pub /etc/apk/keys/ || true
cp "$HERE/mkimg.routier.sh"      "$APORTS/scripts/mkimg.routier.sh"
cp "$HERE/genapkovl-routier.sh" "$APORTS/scripts/genapkovl-routier.sh"

mkdir -p "$OUTDIR"
chown builder: "$OUTDIR"

ROUTIER_BIN=""
BINDIR=""
if [ "$FLAVOR" = "installer" ]; then
	ROUTIER_APK=$(ls -t "$REPO/$ARCH"/routier-[0-9]*.apk 2>/dev/null | head -1)
	[ -n "$ROUTIER_APK" ] || { echo "no routier apk in $REPO/$ARCH, run task apk first" >&2; exit 1; }
	BINDIR="$OUTDIR/.installer-bin"
	rm -rf "$BINDIR"
	mkdir -p "$BINDIR"
	OPENRC_APK=$(ls -t "$REPO/$ARCH"/routier-openrc-[0-9]*.apk 2>/dev/null | head -1)
	[ -n "$OPENRC_APK" ] || { echo "no routier-openrc apk in $REPO/$ARCH, run task apk first" >&2; exit 1; }
	tar -xzf "$ROUTIER_APK" -C "$BINDIR" usr/bin/routier usr/bin/routier-ui
	tar -xzf "$OPENRC_APK" -C "$BINDIR" etc/init.d/routier-ui etc/init.d/routier-live
	ROUTIER_BIN="$BINDIR/usr/bin/routier"
	chmod -R a+rX "$BINDIR"
fi

su builder -c "export FLAVOR='$FLAVOR'; export OFFLINE='$OFFLINE'; export ROUTIER_REPO_URL='$ROUTIER_REPO_URL'; export ROUTIER_CHANNEL='$ROUTIER_CHANNEL'; export ARCH='$ARCH'; export ROUTIER_BIN='$ROUTIER_BIN'; export ROUTIER_FILES='$BINDIR'; . /home/builder/.abuild/abuild.conf && cd '$APORTS/scripts' && $APORTS/scripts/mkimage.sh \
	--tag '${VERSION}-r${RELEASE}' \
	--outdir '$OUTDIR' \
	--arch '$ARCH' \
	--hostkeys \
	$REPOS \
	--profile $PROFILE"

rm -rf "$OUTDIR/.installer-bin"
chown -R 0:0 "$OUTDIR"

echo "ISO written to $OUTDIR"

KERNEL=$FLAVOR
[ "$FLAVOR" = "installer" ] && KERNEL=lts
ISO=$(ls -t "$OUTDIR/$IMAGE"-*-"$ARCH".iso 2>/dev/null | head -1)
if [ -n "$ISO" ]; then
	BOOTDIR="$HERE/../dist/boot"
	mkdir -p "$BOOTDIR"
	xorriso -osirrox on -indev "$ISO" \
		-extract "/boot/vmlinuz-$KERNEL"   "$BOOTDIR/vmlinuz-$KERNEL" \
		-extract "/boot/initramfs-$KERNEL" "$BOOTDIR/initramfs-$KERNEL" \
		2>/dev/null
	echo "kernel/initramfs extracted to $BOOTDIR"
fi
