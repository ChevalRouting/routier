#!/bin/sh
set -eu

if [ $# -lt 2 ]; then
	echo 'usage: bench.sh user@loopback user@loopback [...]' >&2
	exit 1
fi

for host in "$@"; do
	case "$host" in
		-*|*[!a-zA-Z0-9_.@:-]*|*@)
			printf 'Invalid host %s: use user@loopback-address.\n' "$host" >&2
			exit 1
			;;
	esac
done

duration=${DURATION:-8}
streams=${STREAMS:-1}
rates=${UDP_RATES:-200M 1G}
port=${PORT:-5201}
length=${UDP_LENGTH:-1350}
for value in "$duration" "$streams" "$port" "$length"; do
	case "$value" in
		''|*[!0-9]*) echo 'DURATION, STREAMS, PORT and UDP_LENGTH must be positive integers.' >&2; exit 1 ;;
	esac
done

for rate in $rates; do
	case "$rate" in
		*[!0-9KMG]*) printf 'Invalid UDP rate %s: use a number with an optional K, M or G suffix.\n' "$rate" >&2; exit 1 ;;
	esac
done

tag=routier-bench
pidfile=/tmp/routier-bench.pid
marker=/tmp/routier-bench.installed

address() {
	printf '%s\n' "${1#*@}"
}

sources() {
	for host in "$@"; do
		printf '%s\n' "$(address "$host")"
	done | paste -sd, -
}

remote() {
	ssh -o BatchMode=yes -- "$@"
}

setup() {
	remote "$1" sh -s -- "$(address "$1")" "$2" "$port" "$tag" "$pidfile" "$marker" <<'EOF'
set -eu
addr=$1 sources=$2 port=$3 tag=$4 pidfile=$5 marker=$6
if ! command -v iperf3 >/dev/null; then
	doas apk add -q iperf3 >/dev/null
	touch "$marker"
fi
printf 'insert rule inet routier input ip saddr { %s } ip daddr %s meta l4proto { tcp, udp } th dport %s accept comment "%s"\n' "$sources" "$addr" "$port" "$tag" > /tmp/routier-bench.nft
doas nft -f /tmp/routier-bench.nft
rm -f /tmp/routier-bench.nft
iperf3 -s -D -B "$addr" -p "$port" -I "$pidfile"
EOF
}

teardown() {
	remote "$1" sh -s -- "$tag" "$pidfile" "$marker" <<'EOF' || true
tag=$1 pidfile=$2 marker=$3
[ -f "$pidfile" ] && kill "$(cat "$pidfile")" 2>/dev/null
rm -f "$pidfile"
for handle in $(doas nft -a list chain inet routier input | grep "comment \"$tag\"" | sed -n 's/.*# handle \([0-9]*\).*/\1/p'); do
	doas nft delete rule inet routier input handle "$handle"
done
if [ -f "$marker" ] && doas apk del -q iperf3 >/dev/null; then
	rm -f "$marker"
fi
EOF
}

hosts="$*"

cleanup() {
	for host in $hosts; do
		teardown "$host"
	done
}

trap cleanup EXIT
trap 'exit 130' INT TERM

allowed=$(sources "$@")
for host in "$@"; do
	setup "$host" "$allowed"
done

printf '%-16s %-16s %-8s %14s  %s\n' FROM TO TEST RESULT DETAIL
for from in "$@"; do
	for to in "$@"; do
		[ "$from" = "$to" ] && continue
		src=$(address "$from")
		dst=$(address "$to")
		remote "$from" iperf3 -c "$dst" -B "$src" -p "$port" -t "$duration" -P "$streams" -f m |
			awk -v from="$src" -v to="$dst" -v streams="x$streams" '
				/sender$/ { retr = $(NF-1) }
				/receiver$/ { rate = $(NF-2) }
				END { printf "%-16s %-16s %-8s %9s Mbit/s  retr=%s\n", from, to, "tcp" streams, rate, retr }'
		for rate in $rates; do
			remote "$from" iperf3 -u -c "$dst" -B "$src" -p "$port" -t "$duration" -b "$rate" -l "$length" -f m |
				awk -v from="$src" -v to="$dst" -v test="udp$rate" '
					/receiver$/ { for (i = 1; i <= NF; i++) { if ($i == "Mbits/sec") rate = $(i-1); if ($i == "ms") jitter = $(i-1) } loss = $(NF-1) }
					END { gsub(/[()]/, "", loss); printf "%-16s %-16s %-8s %9s Mbit/s  loss=%s jitter=%sms\n", from, to, test, rate, loss, jitter }'
		done
	done
done
