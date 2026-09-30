#!/bin/sh
set -eu

HERE=$(cd "$(dirname "$0")/.." && pwd)
ARCH=$(uname -m)
FLAVOR="${FLAVOR:-lts}"
DISK_IMG="$HERE/dist/disk.img"
KERNEL="$HERE/dist/boot/vmlinuz-$FLAVOR"
INITRD="$HERE/dist/boot/initramfs-$FLAVOR"

case "$ARCH" in
	aarch64) CONSOLE="ttyAMA0" ;;
	*)       CONSOLE="ttyS0"   ;;
esac
APPEND="root=/dev/routier/root rootfstype=ext4 console=${CONSOLE},115200 quiet"

[ -f "$KERNEL" ]   || { echo "no kernel at $KERNEL, run task iso first" >&2; exit 1; }
[ -f "$INITRD" ]   || { echo "no initramfs at $INITRD, run task iso first" >&2; exit 1; }
[ -f "$DISK_IMG" ] || { echo "no disk at $DISK_IMG, run task run-iso to install first" >&2; exit 1; }

case "$ARCH" in
	aarch64)
		exec qemu-system-aarch64 \
			-machine virt \
			-enable-kvm \
			-cpu host \
			-m 1G \
			-smp 2 \
			-kernel "$KERNEL" \
			-initrd "$INITRD" \
			-append "$APPEND" \
			-drive file="$DISK_IMG",if=none,id=hd0,format=qcow2 \
			-device virtio-scsi-pci \
			-device scsi-hd,drive=hd0 \
			-netdev user,id=net0,hostfwd=tcp::2222-:22 \
			-device virtio-net-pci,netdev=net0 \
			-nographic -serial mon:stdio
		;;
	x86_64)
		exec qemu-system-x86_64 \
			-machine q35 -enable-kvm -cpu host \
			-m 1G -smp 2 \
			-kernel "$KERNEL" \
			-initrd "$INITRD" \
			-append "$APPEND" \
			-drive file="$DISK_IMG",if=none,id=hd0,format=qcow2 \
			-device ich9-ahci,id=ahci \
			-device ide-hd,drive=hd0,bus=ahci.0 \
			-nographic -serial mon:stdio \
			-netdev user,id=net0,hostfwd=tcp::2222-:22 \
			-device virtio-net-pci,netdev=net0
		;;
	*)
		echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac
