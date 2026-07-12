#!/bin/sh -e

HOSTNAME="${1:-routier}"

cleanup() { rm -rf "$tmp"; }
makefile() {
	OWNER="$1"; PERMS="$2"; FILENAME="$3"
	cat >"$FILENAME"
	chown "$OWNER" "$FILENAME"
	chmod "$PERMS" "$FILENAME"
}
rc_add() {
	mkdir -p "$tmp/etc/runlevels/$2"
	ln -sf "/etc/init.d/$1" "$tmp/etc/runlevels/$2/$1"
}

tmp="$(mktemp -d)"
trap cleanup EXIT

mkdir -p "$tmp/etc"
makefile root:root 0644 "$tmp/etc/hostname" <<EOF
$HOSTNAME
EOF

makefile root:root 0644 "$tmp/etc/modules" <<EOF
ext4
vfat
dm-mod
EOF

: ${ARCH:=$(uname -m)}
case "$ARCH" in
	aarch64) SERIAL_TTY=ttyAMA0 ;;
	*)        SERIAL_TTY=ttyS0   ;;
esac

makefile root:root 0644 "$tmp/etc/inittab" <<EOF
# /etc/inittab

::sysinit:/sbin/openrc sysinit
::sysinit:/sbin/openrc boot
::wait:/sbin/openrc default

tty1::respawn:/sbin/getty 38400 tty1
tty2::respawn:/sbin/getty 38400 tty2
tty3::respawn:/sbin/getty 38400 tty3
tty4::respawn:/sbin/getty 38400 tty4
tty5::respawn:/sbin/getty 38400 tty5
tty6::respawn:/sbin/getty 38400 tty6

${SERIAL_TTY}::respawn:/sbin/getty -L 115200 ${SERIAL_TTY} vt100

::ctrlaltdel:/sbin/reboot

::shutdown:/sbin/openrc shutdown
EOF

mkdir -p "$tmp/etc/apk"
makefile root:root 0644 "$tmp/etc/apk/world" <<EOF
linux-virt
alpine-base
tzdata
routier
routier-openrc
openssh
lsblk
parted
lvm2
lvm2-openrc
e2fsprogs
dosfstools
wipefs
grub-efi
efibootmgr
htop
tcpdump
mtr
util-linux-openrc
EOF

rc_add devfs sysinit
rc_add dmesg sysinit
rc_add mdev sysinit
rc_add hwdrivers sysinit
rc_add modloop sysinit

rc_add modules boot
rc_add sysctl boot
rc_add hostname boot
rc_add bootmisc boot
rc_add syslog boot

rc_add routier-net boot
rc_add routier-firstboot boot
rc_add sshd default
rc_add routier default
rc_add routier-ui default
rc_add cronie default

rc_add mount-ro shutdown
rc_add killprocs shutdown
rc_add savecache shutdown

tar -c -C "$tmp" etc | gzip -9n >"$HOSTNAME.apkovl.tar.gz"
