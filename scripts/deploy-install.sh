#!/bin/sh
set -eu

arch=${1:?architecture required}
version=${2:?package version required}
if [ "$(id -u)" != 0 ]; then
	if command -v doas >/dev/null 2>&1; then
		exec doas sh "$0" "$@"
	elif command -v sudo >/dev/null 2>&1; then
		exec sudo sh "$0" "$@"
	fi
	echo 'Deploy as root or install doas/sudo for privilege elevation.' >&2
	exit 1
fi

cd -- "$(dirname -- "$0")"
[ "$(apk --print-arch)" = "$arch" ] || { echo 'Host architecture changed; refusing install.' >&2; exit 1; }
if apk info -e "routier=$version" >/dev/null 2>&1; then
	echo 'This version is already installed. Bump VERSION or RELEASE to deploy a new build.' >&2
	exit 1
fi

install -d -m 755 /etc/apk/keys
install -m 644 routier.rsa.pub /etc/apk/keys/routier.rsa.pub
apk add --upgrade ./routier.apk ./routier-openrc.apk
for package in routier routier-openrc; do
	if ! apk info -e "$package=$version" >/dev/null 2>&1; then
		printf 'Expected %s-%s after installation.\n' "$package" "$version" >&2
		exit 1
	fi
done
routier version
