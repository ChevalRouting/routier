#!/bin/sh
set -eu

host=${1:?usage: deploy.sh user@host version release}
version=${2:?version required}
release=${3:?release required}
case "$host" in
	-*|*[!a-zA-Z0-9_.@:\[\]-]*)
		echo 'Use an SSH hostname, user@host, or SSH config alias for HOST.' >&2
		exit 1
		;;
esac
case "$version" in
	''|*[!a-zA-Z0-9._+~-]*) echo 'Invalid package version.' >&2; exit 1 ;;
esac
case "$release" in
	''|*[!0-9]*) echo 'RELEASE must be a nonnegative integer.' >&2; exit 1 ;;
esac

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
arch=$(ssh -- "$host" 'apk --print-arch')
case "$arch" in
	x86_64|aarch64) ;;
	*) printf 'Unsupported Alpine architecture: %s\n' "$arch" >&2; exit 1 ;;
esac

package="$root/dist/$arch/routier-$version-r$release.apk"
openrc="$root/dist/$arch/routier-openrc-$version-r$release.apk"
if [ ! -f "$package" ] || [ ! -f "$openrc" ]; then
	"${TASK_BIN:-task}" --dir "$root" apk "ARCH=$arch" "VERSION=$version" "RELEASE=$release" \
		"CONTAINER=${CONTAINER:-docker}" "LOCAL_ANYK=${LOCAL_ANYK:-0}" "PODMAN_TTY=${PODMAN_TTY:-}"
else
	printf 'Using existing Routier %s-r%s packages for %s\n' "$version" "$release" "$arch"
fi

for file in "$package" "$openrc" "$root/routier.rsa.pub"; do
	[ -f "$file" ] || { printf 'Missing deployment artifact: %s\n' "$file" >&2; exit 1; }
done

remote_dir=$(ssh -- "$host" 'mktemp -d /tmp/routier-deploy.XXXXXXXX')
case "$remote_dir" in
	/tmp/routier-deploy.*)
		suffix=${remote_dir#/tmp/routier-deploy.}
		case "$suffix" in
			''|*[!a-zA-Z0-9]*) echo 'Invalid remote staging directory.' >&2; exit 1 ;;
		esac
		;;
	*) echo 'Invalid remote staging directory.' >&2; exit 1 ;;
esac

complete=false
finish() {
	if [ "$complete" != true ]; then
		printf 'Deployment failed; staged files remain at %s:%s\n' "$host" "$remote_dir" >&2
	fi
}
trap finish EXIT

printf 'Uploading Routier %s-r%s for %s to %s\n' "$version" "$release" "$arch" "$host"
scp -- "$package" "$host:$remote_dir/routier.apk"
scp -- "$openrc" "$host:$remote_dir/routier-openrc.apk"
scp -- "$root/routier.rsa.pub" "$root/scripts/deploy-install.sh" "$host:$remote_dir/"
ssh -t -- "$host" "sh '$remote_dir/deploy-install.sh' '$arch' '$version-r$release'"
ssh -- "$host" "rm -rf -- '$remote_dir'"
complete=true
printf 'Routier %s-r%s installed on %s.\n' "$version" "$release" "$host"
