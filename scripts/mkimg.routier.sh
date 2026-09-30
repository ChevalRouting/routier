_profile_routier_common() {
	local flavor="$1"
	title="Routier"
	desc="Routier router/firewall appliance"
	profile_base
	hostname="routier"
	image_name="routier"
	image_ext="iso"
	output_format="iso"
	arch="aarch64 x86 x86_64"
	kernel_flavors="$flavor"
	kernel_addons=""
	initfs_features="ata base cdrom ext4 keymap lvm nvme scsi squashfs usb virtio xfs"
	initfs_cmdline="modules=loop,squashfs,cdrom quiet"
	apks="$(echo "$apks" | tr ' ' '\n' | grep -v '^network-extras$' | tr '\n' ' ')linux-$flavor htop mtr tcpdump tzdata routier routier-openrc lsblk parted lvm2 lvm2-openrc xfsprogs dosfstools wipefs grub-efi efibootmgr util-linux-openrc agetty"
	apkovl="genapkovl-routier.sh"

	grub_gen_config() {
		cat <<-EOF
		set default=0
		set timeout=3

		menuentry "Routier" {
		    linux  /boot/vmlinuz-$kernel_flavors $initfs_cmdline $kernel_cmdline
		    initrd /boot/initramfs-$kernel_flavors
		}
		EOF
	}
}

profile_routier() {
	_profile_routier_common lts
}

profile_routier_virt() {
	_profile_routier_common virt
	image_name="routier-virt"
}

profile_routier_stable() {
	_profile_routier_common stable
	image_name="routier-stable"
}
