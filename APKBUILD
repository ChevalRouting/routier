# Maintainer: ChevalRouting <chevalrouting@protonmail.com>
pkgname=routier
pkgver=${VERSION}
pkgrel=${RELEASE}
pkgdesc="Declarative router/firewall manager with web UI (BGP/OSPF, nftables, WireGuard)"
url="https://github.com/ChevalRouting/routier"
arch="all"
license="MIT"
depends="nftables frr wireguard-tools iproute2 dhcpcd cronie frr-pythontools openssl chrony keepalived conntrack-tools zstd rrdtool radvd kea-dhcp4 kea-dhcp6 kea-hook-lease-cmds mtr traceroute bind-tools curl nmap"
makedepends="go nodejs npm"
options="net"
install="$pkgname.post-install $pkgname.post-upgrade $pkgname.pre-deinstall"
subpackages="$pkgname-openrc"
source="$pkgname-$pkgver.tar.gz"

build() {
	unset GOCACHE
	unset GOMODCACHE
	unset GOTMPDIR

	( cd web && npm ci && npm run build )

	_ldflags="-s -w -X main.VERSION=${pkgver}-r${pkgrel} -X github.com/ChevalRouting/routier/pkg/api/app.Version=${pkgver}-r${pkgrel}"
	CGO_ENABLED=0 go build -tags prod -ldflags="$_ldflags" -o routier ./cmd/routier
	CGO_ENABLED=0 go build -tags prod -ldflags="$_ldflags" -o routier-ui ./cmd/routier-ui
}

check() {
	go vet ./...
}

package() {
	# routier is installed setuid-root, group wheel, mode 4750: the root daemon
	# runs it as root, members of wheel (the routier admin user) escalate through
	# it, and other users cannot run it at all.
	install -Dm755 routier "$pkgdir/usr/bin/routier"
	chgrp wheel "$pkgdir/usr/bin/routier"
	chmod 4750 "$pkgdir/usr/bin/routier"
	install -Dm755 routier-ui "$pkgdir/usr/bin/routier-ui"
	# Convenience symlink so admins can run setup from PATH on the live ISO.
	install -dm755 "$pkgdir/usr/sbin"
	ln -s /usr/bin/routier "$pkgdir/usr/sbin/routier-setup"
	# doas rule: routier user may run anything as root (manages network, nftables, etc.)
	install -Dm640 scripts/routier.doas "$pkgdir/etc/doas.d/routier.conf"
	install -Dm644 scripts/routier.crontab "$pkgdir/etc/cron.d/routier"
	install -Dm644 scripts/routier.profile "$pkgdir/etc/profile.d/routier.sh"
	install -dm750 "$pkgdir/etc/routier"
	install -dm755 "$pkgdir/etc/routier/templates"
	install -dm755 "$pkgdir/etc/nftables.d"
}

openrc() {
	depends="$pkgname=$pkgver-r$pkgrel openrc util-linux-openrc"
	install_if="$pkgname=$pkgver-r$pkgrel openrc"
	install -Dm755 "$builddir/scripts/routier-net.initd" "$subpkgdir/etc/init.d/routier-net"
	install -Dm755 "$builddir/scripts/routier.initd" "$subpkgdir/etc/init.d/routier"
	install -Dm755 "$builddir/scripts/routier-ui.initd" "$subpkgdir/etc/init.d/routier-ui"
	install -Dm755 "$builddir/scripts/routier-firstboot.initd" "$subpkgdir/etc/init.d/routier-firstboot"
}
