#!/bin/sh

set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
ARCH="$(apk --print-arch)"
VERSION="${VERSION:-0.0.0}"
RELEASE="${RELEASE:-0}"
REPO="${ROUTIER_REPO:-$HERE/../dist}"
OUTDIR="${OUTDIR:-$HERE/../dist/cloud}"
SIZE="${SIZE:-2G}"
KFLAVOR="${KFLAVOR:-virt}"
ROUTIER_REPO_URL="${ROUTIER_REPO_URL-https://repo.routier.cloud}"
ROUTIER_CHANNEL="${ROUTIER_CHANNEL:-nightly}"
MIRROR="${ALPINE_MIRROR:-https://dl-cdn.alpinelinux.org/alpine}"
BRANCH="v$(cut -d. -f1,2 /etc/alpine-release)"
IMAGE="$OUTDIR/routier-cloud-${VERSION}-r${RELEASE}-${ARCH}.qcow2"

case "$ARCH" in
	aarch64) GRUB_TARGET=arm64-efi;  CONSOLE=ttyAMA0 ;;
	x86_64)  GRUB_TARGET=x86_64-efi; CONSOLE=ttyS0 ;;
	*) echo "unsupported arch $ARCH" >&2; exit 1 ;;
esac

[ -f "$REPO/$ARCH/APKINDEX.tar.gz" ] || { echo "no signed repo at $REPO/$ARCH, run task apk first" >&2; exit 1; }

apk add --no-cache parted e2fsprogs dosfstools util-linux qemu-img >/dev/null
install -m 644 "$HERE/../routier.rsa.pub" /etc/apk/keys/routier.rsa.pub

work=$(mktemp -d)
mnt="$work/mnt"
raw="$work/disk.raw"
esp=""
root=""

cleanup() {
	set +e
	for m in "$mnt/dev" "$mnt/proc" "$mnt/sys" "$mnt/boot/efi" "$mnt"; do
		mountpoint -q "$m" && umount "$m"
	done
	[ -n "$esp" ] && losetup -d "$esp"
	[ -n "$root" ] && losetup -d "$root"
	rm -rf "$work"
}
trap cleanup EXIT

partition_loop() {
	line=$(parted -s -m "$raw" unit B print | grep "^$1:")
	offset=$(echo "$line" | cut -d: -f2 | tr -d B)
	size=$(echo "$line" | cut -d: -f4 | tr -d B)
	dev=$(losetup -f)
	[ -e "$dev" ] || mknod "$dev" b 7 "${dev#/dev/loop}"
	losetup -o "$offset" --sizelimit "$size" "$dev" "$raw"
	echo "$dev"
}

truncate -s "$SIZE" "$raw"
parted -s "$raw" mklabel gpt \
	mkpart ESP fat32 1MiB 257MiB set 1 esp on \
	mkpart root ext4 257MiB 100%
esp=$(partition_loop 1)
root=$(partition_loop 2)

mkfs.vfat -F32 -n EFI "$esp" >/dev/null
mkfs.ext4 -q -L root "$root"
mkdir -p "$mnt"
mount "$root" "$mnt"
mkdir -p "$mnt/boot/efi" "$mnt/dev" "$mnt/proc" "$mnt/sys"
mount "$esp" "$mnt/boot/efi"
mount --bind /dev "$mnt/dev"
mount -t proc proc "$mnt/proc"
mount -t sysfs sys "$mnt/sys"

mkdir -p "$mnt/etc/apk/keys" "$mnt/etc/mkinitfs"
cp /etc/apk/keys/* "$mnt/etc/apk/keys/"
printf '%s/%s/main\n%s/%s/community\n' "$MIRROR" "$BRANCH" "$MIRROR" "$BRANCH" > "$mnt/etc/apk/repositories"
[ -z "$ROUTIER_REPO_URL" ] || printf '%s/%s\n' "$ROUTIER_REPO_URL" "$ROUTIER_CHANNEL" >> "$mnt/etc/apk/repositories"
echo 'features="ata base ext4 keymap nvme scsi usb virtio"' > "$mnt/etc/mkinitfs/mkinitfs.conf"

apk --root "$mnt" --initdb --quiet --repository "$REPO" add \
	alpine-base "linux-$KFLAVOR" grub-efi openssh chrony tzdata e2fsprogs e2fsprogs-extra \
	cloud-init cloud-utils-growpart routier routier-openrc doas

root_uuid=$(blkid -s UUID -o value "$root")
esp_uuid=$(blkid -s UUID -o value "$esp")

cat > "$mnt/etc/fstab" <<EOF
UUID=$root_uuid / ext4 defaults,noatime 0 1
UUID=$esp_uuid /boot/efi vfat defaults 0 2
EOF

echo routier > "$mnt/etc/hostname"
printf '127.0.0.1\tlocalhost localhost.localdomain\n::1\tlocalhost localhost.localdomain\n' > "$mnt/etc/hosts"
printf 'auto lo\niface lo inet loopback\n' > "$mnt/etc/network/interfaces"

cat > "$mnt/etc/inittab" <<EOF
::sysinit:/sbin/openrc sysinit
::sysinit:/sbin/openrc boot
::wait:/sbin/openrc default

tty1::respawn:/sbin/getty 38400 tty1
$CONSOLE::respawn:/sbin/getty -L 115200 $CONSOLE vt100

::ctrlaltdel:/sbin/reboot
::shutdown:/sbin/openrc shutdown
EOF

for svc in devfs dmesg mdev hwdrivers; do chroot "$mnt" rc-update add "$svc" sysinit >/dev/null; done
for svc in modules sysctl hostname bootmisc syslog networking; do chroot "$mnt" rc-update add "$svc" boot >/dev/null; done
chroot "$mnt" rc-update add routier-net boot >/dev/null
for svc in sshd chronyd cronie routier routier-ui routier-cloud; do chroot "$mnt" rc-update add "$svc" default >/dev/null; done
for svc in mount-ro killprocs savecache; do chroot "$mnt" rc-update add "$svc" shutdown >/dev/null; done
chroot "$mnt" setup-cloud-init >/dev/null
cat > "$mnt/etc/cloud/cloud.cfg.d/90-routier.cfg" <<EOF
system_info:
  default_user:
    name: alpine
    lock_passwd: false
    passwd: "*"
    groups: [wheel]
EOF
echo "permit nopass alpine" > "$mnt/etc/doas.d/alpine.conf"

kver=$(ls "$mnt/lib/modules" | sort | tail -1)
chroot "$mnt" mkinitfs "$kver" >/dev/null
chroot "$mnt" grub-install --target="$GRUB_TARGET" --efi-directory=/boot/efi --boot-directory=/boot --removable --no-nvram --modules="part_gpt ext2 fat" >/dev/null
chroot "$mnt" routier grub-config "$KFLAVOR" --root-args "root=UUID=$root_uuid rootfstype=ext4" > "$mnt/boot/grub/grub.cfg"

rm -rf "$mnt/var/cache/apk/"* "$mnt/etc/ssh/ssh_host_"* "$mnt/var/lib/routier/tls."*
umount "$mnt/dev" "$mnt/proc" "$mnt/sys" "$mnt/boot/efi" "$mnt"
losetup -d "$esp"
losetup -d "$root"
esp=""
root=""

mkdir -p "$OUTDIR"
qemu-img convert -c -f raw -O qcow2 "$raw" "$IMAGE"
echo "cloud image written to $IMAGE"
