#!/bin/sh
set -eu

HERE=$(cd "$(dirname "$0")/.." && pwd)
ISO_DIR="$HERE/dist/iso"
DISK_IMG="$HERE/dist/disk.img"

ISO=""
if [ $# -ge 1 ] && [ -n "$1" ]; then
	ISO="$1"
	[ -f "$ISO" ] || { echo "not found: $ISO" >&2; exit 1; }
elif [ $# -eq 0 ]; then
	FOUND=$(ls -t "$ISO_DIR"/*.iso 2>/dev/null | head -1)
	[ -n "$FOUND" ] && ISO="$FOUND"
fi

if [ ! -f "$DISK_IMG" ]; then
	echo "creating $DISK_IMG (10G)..." >&2
	qemu-img create -f qcow2 "$DISK_IMG" 10G
fi

if [ -n "$ISO" ]; then
	case "$ISO" in
		*x86_64*)  ARCH=x86_64  ;;
		*)         ARCH=$(uname -m) ;;
	esac
else
	ARCH=$(uname -m)
fi

case "$ARCH" in
	aarch64)
		CDROM_ARGS=""
		if [ -n "$ISO" ]; then
			CDROM_ARGS="-drive file=$ISO,if=none,id=cdrom0,media=cdrom,readonly=on -device scsi-cd,drive=cdrom0"
		fi
		exec qemu-system-aarch64 \
			-machine virt \
			-enable-kvm \
			-cpu host \
			-m 1G \
			-smp 2 \
			-bios /usr/share/edk2/aarch64/QEMU_EFI.fd \
			-drive file="$DISK_IMG",if=none,id=hd0,format=qcow2 \
			-device virtio-scsi-pci \
			$CDROM_ARGS \
			-device scsi-hd,drive=hd0 \
			-device virtio-gpu \
			-device qemu-xhci,id=xhci \
			-device usb-kbd,bus=xhci.0 \
			-device usb-mouse,bus=xhci.0 \
			-netdev tap,id=net0,ifname=tap0,script=no,downscript=no \
			-device virtio-net-pci,netdev=net0 \
			-serial mon:stdio
		;;
	arm64)
		CDROM_ARGS=""
		if [ -n "$ISO" ]; then
			CDROM_ARGS="-drive file=$ISO,if=none,id=cdrom0,media=cdrom,readonly=on -device scsi-cd,drive=cdrom0"
		fi
		exec qemu-system-aarch64 \
			-machine virt \
			-accel hvf \
			-cpu host \
			-m 1G \
			-smp 2 \
			-bios /opt/homebrew/share/qemu/edk2-aarch64-code.fd \
			-drive file="$DISK_IMG",if=none,id=hd0,format=qcow2 \
			-device virtio-scsi-pci \
			$CDROM_ARGS \
			-device scsi-hd,drive=hd0 \
			-device virtio-gpu \
			-device qemu-xhci,id=xhci \
			-device usb-kbd,bus=xhci.0 \
			-device usb-mouse,bus=xhci.0 \
			-netdev vmnet-shared,id=net0 \
			-device virtio-net-pci,netdev=net0 \
			-serial mon:stdio
		;;
	x86_64)
		CDROM_ARGS=""
		[ -n "$ISO" ] && CDROM_ARGS="-cdrom $ISO"
		exec qemu-system-x86_64 \
			-machine q35 -enable-kvm -cpu host \
			-m 1G -smp 2 \
			$CDROM_ARGS \
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
