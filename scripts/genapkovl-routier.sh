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

makefile root:root 0644 "$tmp/etc/hosts" <<EOF
127.0.0.1	localhost localhost.localdomain $HOSTNAME
::1	localhost localhost.localdomain $HOSTNAME
EOF

makefile root:root 0644 "$tmp/etc/modules" <<EOF
xfs
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

tty1::respawn:/sbin/agetty --autologin root --noclear tty1 linux
tty2::respawn:/sbin/agetty --autologin routier --noclear tty2 linux
tty3::respawn:/sbin/getty 38400 tty3
tty4::respawn:/sbin/getty 38400 tty4
tty5::respawn:/sbin/getty 38400 tty5
tty6::respawn:/sbin/getty 38400 tty6

# ${SERIAL_TTY}::respawn:/bin/sh -c 'test -e /dev/${SERIAL_TTY} && exec /sbin/getty -L 115200 ${SERIAL_TTY} vt100 || sleep 2147483647'

::ctrlaltdel:/sbin/reboot

::shutdown:/sbin/openrc shutdown
EOF

mkdir -p "$tmp/etc/apk"

: ${ALPINE_MIRROR:=https://dl-cdn.alpinelinux.org/alpine}
branch="$(cut -d. -f1,2 /etc/alpine-release 2>/dev/null || true)"
if [ -n "$branch" ]; then
	branch="v$branch"
else
	branch="edge"
fi
makefile root:root 0644 "$tmp/etc/apk/repositories" <<EOF
$ALPINE_MIRROR/$branch/main
$ALPINE_MIRROR/$branch/community
EOF

mkdir -p "$tmp/etc/apk/keys"
if [ -f /etc/apk/keys/routier.rsa.pub ]; then
	install -m 0644 /etc/apk/keys/routier.rsa.pub "$tmp/etc/apk/keys/routier.rsa.pub"
fi

mkdir -p "$tmp/etc/xdg/nvim"
makefile root:root 0644 "$tmp/etc/xdg/nvim/sysinit.vim" <<EOF
set number
set expandtab
set tabstop=4
set shiftwidth=4
set softtabstop=4
set smartindent
set mouse=
set termguicolors

augroup routier_yaml
	autocmd!
	autocmd FileType yaml setlocal expandtab tabstop=4 shiftwidth=4 softtabstop=4 indentkeys=!^F,o,O
augroup END
EOF

: ${FLAVOR:=lts}
makefile root:root 0644 "$tmp/etc/apk/world" <<EOF
linux-$FLAVOR
alpine-base
tzdata
routier
routier-openrc
openssh
lsblk
parted
lvm2
lvm2-openrc
xfsprogs
dosfstools
wipefs
grub-efi
efibootmgr
htop
tcpdump
mtr
neovim
util-linux-openrc
agetty
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
