profile_routier() {
	title="Routier"
	desc="Routier router/firewall appliance"
	profile_base
	image_ext="iso"
	output_format="iso"
	arch="aarch64 x86 x86_64"
	kernel_flavors="virt"
	kernel_addons=""
	initfs_features="ata base cdrom ext4 keymap lvm nvme scsi squashfs usb virtio"
	initfs_cmdline="modules=loop,squashfs,cdrom quiet"
	apks="$(echo "$apks" | tr ' ' '\n' | grep -v '^network-extras$' | tr '\n' ' ')linux-virt htop mtr tcpdump tzdata routier routier-openrc lsblk parted lvm2 lvm2-openrc e2fsprogs dosfstools wipefs grub-efi efibootmgr util-linux-openrc"
	apkovl="genapkovl-routier.sh"

	grub_gen_config() {
		cat <<-EOF
		set default=0
		set timeout=3

		menuentry "Routier" {
		    linux  /boot/vmlinuz-virt $initfs_cmdline $kernel_cmdline
		    initrd /boot/initramfs-virt
		}
		EOF
	}
}
