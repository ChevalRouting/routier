FROM alpine:3.23

RUN apk update && \
  apk add alpine-sdk nftables frr frr-pythontools frr-openrc wireguard-tools iproute2 make go dhcpcd cronie nodejs npm chrony keepalived conntrack-tools s6 zstd rrdtool icu-data-full radvd kea kea-dhcp4 kea-dhcp6 kea-hook-lease-cmds mtr traceroute bind-tools curl nmap \
  alpine-conf xorriso squashfs-tools mtools dosfstools grub git && \
  adduser -D builder && \
  addgroup builder abuild

RUN mkdir /work && chown builder:builder /work

USER builder
WORKDIR /home/builder
COPY --chown=builder:builder go.mod go.sum /home/builder/

RUN go mod download

USER root
