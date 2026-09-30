#!/bin/sh
set -e

uid="${ROUTIER_HOST_UID:-1000}"
gid="${ROUTIER_HOST_GID:-1000}"
group="$(getent group "$gid" | cut -d: -f1)"

if [ -z "$group" ]; then
	group=routier
	addgroup -g "$gid" -S "$group"
fi

if ! getent passwd "$uid" >/dev/null 2>&1; then
	adduser -S -D -H -h /tmp/routier-home -u "$uid" -G "$group" routier
fi

export HOME=/tmp/routier-home
mkdir -p "$HOME" /go/pkg/mod /go/build-cache /npm-cache
chown -R "$uid:$gid" "$HOME" /go/pkg /go/build-cache /npm-cache

exec su-exec "$uid:$gid" "$@"
