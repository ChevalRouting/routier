#!/bin/sh
set -e

: "${container:=docker}"
export container

mkdir -p /run/openrc
touch /run/openrc/softlevel

if ! mountpoint -q /sys/fs/cgroup 2>/dev/null; then
    mount -t cgroup2 none /sys/fs/cgroup 2>/dev/null || true
fi

if command -v openrc-init >/dev/null 2>&1; then
    exec openrc-init default
fi

openrc sysinit
openrc boot
openrc default

exec tail -F /var/log/routier-ui.log 2>/dev/null
