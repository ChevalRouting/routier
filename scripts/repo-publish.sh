#!/bin/sh
set -eu

usage='usage: REPO_TARGET=user@host:/srv/routier/repo VERSION=x.y.z [RELEASE=n] [CHANNEL=nightly|stable] repo-publish.sh'

publish_inside() {
	apk add --no-cache abuild openssl rsync openssh-client >/dev/null
	mkdir -p /root/.abuild
	install -m 600 /key/routier.rsa /root/.abuild/routier.rsa
	install -m 644 /key/routier.rsa.pub /root/.abuild/routier.rsa.pub
	echo 'PACKAGER_PRIVKEY="/root/.abuild/routier.rsa"' > /root/.abuild/abuild.conf
	cp /key/routier.rsa.pub /etc/apk/keys/

	remote="$TARGET/$CHANNEL/$ARCH/"
	stage=$(mktemp -d)
	if ! rsync -a --include='*.apk' --exclude='*' -e "$RSH" "$remote" "$stage/" 2>/tmp/pull.err; then
		rsync --list-only -e "$RSH" "$TARGET/" >/dev/null || { cat /tmp/pull.err >&2; echo "cannot reach $TARGET" >&2; exit 1; }
		echo "starting the $CHANNEL/$ARCH channel"
	fi

	for pkg in "routier-$VERSION-r$RELEASE.apk" "routier-openrc-$VERSION-r$RELEASE.apk"; do
		cp "/pkgs/$pkg" "$stage/$pkg"
		touch "$stage/$pkg"
	done

	if [ "$KEEP_DAYS" -gt 0 ]; then
		find "$stage" -maxdepth 1 -type f -name '*.apk' -mtime +"$KEEP_DAYS" -print -delete | sed 's|.*/|pruned |'
	fi

	cd "$stage"
	set -- *.apk
	if ! apk index --rewrite-arch "$ARCH" -o APKINDEX.tar.gz "$@" 2>/tmp/index.err; then
		cat /tmp/index.err >&2
		echo "apk index failed, nothing was published" >&2
		exit 1
	fi

	awk '/^WARNING: No provider/{s=1;next} s&&/^[[:space:]]/{next} {s=0} /^WARNING: Total of .*unsatisfiable/{next} /Your repository may be broken/{next} {print}' /tmp/index.err >&2 || true
	got=$(tar -Oxzf APKINDEX.tar.gz APKINDEX | grep -c '^P:')
	[ "$got" = "$#" ] || { echo "index describes $got of $# packages, nothing was published" >&2; exit 1; }

	abuild-sign -q APKINDEX.tar.gz
	tar -tzf APKINDEX.tar.gz | grep -q '^\.SIGN\.RSA\.routier\.rsa\.pub$' || { echo "index signature missing, nothing was published" >&2; exit 1; }

	rsync -a --mkpath -e "$RSH" --include='*.apk' --exclude='*' "$stage/" "$remote"
	rsync -a -e "$RSH" /key/routier.rsa.pub "$TARGET/routier.rsa.pub"
	rsync -a -e "$RSH" "$stage/APKINDEX.tar.gz" "$remote"
	rsync -a --delete -e "$RSH" --include='*.apk' --include='APKINDEX.tar.gz' --exclude='*' "$stage/" "$remote"
	echo "published routier $VERSION-r$RELEASE to $CHANNEL/$ARCH ($# packages)"
}

if [ "${1:-}" = "--inside" ]; then
	publish_inside
	exit 0
fi

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
TARGET="${REPO_TARGET:?$usage}"
VERSION="${VERSION:?$usage}"
RELEASE="${RELEASE:-0}"
CHANNEL="${CHANNEL:-nightly}"
ARCH="${ARCH:-$(uname -m | sed s/arm64/aarch64/)}"
SIGN_KEY="${SIGN_KEY:-$root/routier.rsa}"
KNOWN_HOSTS="${KNOWN_HOSTS:-$HOME/.ssh/known_hosts}"
IMAGE="${ALPINE_IMAGE:-docker.io/library/alpine:3.23}"
CONTAINER="${CONTAINER:-$(command -v podman >/dev/null 2>&1 && echo podman || echo docker)}"

case "$CHANNEL" in
	nightly) KEEP_DAYS="${KEEP_DAYS:-30}" ;;
	stable) KEEP_DAYS="${KEEP_DAYS:-0}" ;;
	*) echo "CHANNEL must be nightly or stable" >&2; exit 1 ;;
esac

case "$ARCH" in
	x86_64|aarch64) ;;
	*) echo "unsupported ARCH=$ARCH" >&2; exit 1 ;;
esac

for f in "$root/dist/$ARCH/routier-$VERSION-r$RELEASE.apk" "$root/dist/$ARCH/routier-openrc-$VERSION-r$RELEASE.apk" "$SIGN_KEY" "$SIGN_KEY.pub"; do
	[ -f "$f" ] || { echo "missing $f (build with: task apk VERSION=$VERSION RELEASE=$RELEASE ARCH=$ARCH)" >&2; exit 1; }
done

set -- -e TARGET -e VERSION -e RELEASE -e CHANNEL -e ARCH -e KEEP_DAYS \
	-v "$root/dist/$ARCH:/pkgs:ro" \
	-v "$SIGN_KEY:/key/routier.rsa:ro" \
	-v "$SIGN_KEY.pub:/key/routier.rsa.pub:ro" \
	-v "$root/scripts/repo-publish.sh:/repo-publish.sh:ro"

case "$TARGET" in
	*:*)
		RSH="ssh -o BatchMode=yes"
		if [ -f "$KNOWN_HOSTS" ]; then
			set -- "$@" -v "$KNOWN_HOSTS:/ssh/known_hosts:ro"
			RSH="$RSH -o UserKnownHostsFile=/ssh/known_hosts"
		fi

		if [ -n "${SSH_KEY:-}" ]; then
			set -- "$@" -v "$SSH_KEY:/ssh/id:ro"
			RSH="$RSH -o IdentitiesOnly=yes -i /ssh/id"
		elif [ "$(uname -s)" = "Darwin" ]; then
			set -- "$@" -v /run/host-services/ssh-auth.sock:/agent.sock -e SSH_AUTH_SOCK=/agent.sock
		elif [ -n "${SSH_AUTH_SOCK:-}" ]; then
			set -- "$@" -v "$SSH_AUTH_SOCK:/agent.sock" -e SSH_AUTH_SOCK=/agent.sock
		fi
		;;
	*)
		mkdir -p "$TARGET"
		set -- "$@" -v "$(CDPATH= cd -- "$TARGET" && pwd):/target"
		TARGET=/target
		RSH=ssh
		;;
esac

export TARGET VERSION RELEASE CHANNEL ARCH KEEP_DAYS RSH
exec "$CONTAINER" run --rm "$@" -e RSH "$IMAGE" sh /repo-publish.sh --inside
