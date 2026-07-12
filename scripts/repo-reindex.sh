#!/bin/sh
set -e

DIR="${1:?usage: repo-reindex.sh <arch-dir> <arch> [days] [privkey]}"
ARCH="${2:?usage: repo-reindex.sh <arch-dir> <arch> [days] [privkey]}"
DAYS="${3:-30}"
KEY="${4:-$HOME/.abuild/routier.rsa}"
IMAGE="${ALPINE_IMAGE:-docker.io/library/alpine:3.23}"

podman run --rm \
	-e ARCH="$ARCH" -e DAYS="$DAYS" \
	-v "$DIR:/repo" \
	-v "$KEY:/key/routier.rsa:ro" \
	-v "$KEY.pub:/key/routier.rsa.pub:ro" \
	"$IMAGE" \
	sh -euc '
		apk add --no-cache abuild openssl >/dev/null
		mkdir -p /root/.abuild
		cp /key/routier.rsa /key/routier.rsa.pub /root/.abuild/
		chmod 600 /root/.abuild/routier.rsa
		chmod 644 /root/.abuild/routier.rsa.pub
		echo "PACKAGER_PRIVKEY=\"/root/.abuild/routier.rsa\"" > /root/.abuild/abuild.conf
		cp /root/.abuild/routier.rsa.pub /etc/apk/keys/
		cd /repo
		find . -maxdepth 1 -type f -name "*.apk" -mtime +"$DAYS" -exec rm -f {} +
		set -- *.apk
		[ -e "$1" ] || { echo "no packages in /repo" >&2; exit 1; }
		apk index --rewrite-arch "$ARCH" -o APKINDEX.tar.gz "$@" 2>&1 \
			| grep -v "WARNING: No provider\|WARNING: Total of.*unsatisfiable\|Your repository may be broken" || true
		abuild-sign APKINDEX.tar.gz
		echo "reindexed /repo ($# packages, pruned >${DAYS}d)"
	'
