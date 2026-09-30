# syntax=docker/dockerfile:1
# Runnable, OpenRC-based routier image for local development and lab testing.
# Not the package builder at the repo root; this one boots routier like a node.
#
# BuildKit cache mounts keep Go modules, the Go build cache, the npm cache and
# the apk cache warm across builds. After one online build (or `task
# dev-prefetch`) the caches are populated and further rebuilds run offline.
# `go mod vendor` (vendor/ present) makes the Go step fully offline regardless.

FROM golang:1.25-alpine AS build

RUN apk add --no-cache nodejs npm git

WORKDIR /src

COPY web/package.json web/package-lock.json ./web/
COPY web/vendor ./web/vendor/
RUN --mount=type=cache,target=/root/.npm \
    cd web && npm ci --prefer-offline --no-audit --no-fund

COPY . .

ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.npm \
    if [ -d vendor ]; then export GOFLAGS=-mod=vendor; else go mod download; fi && \
    cd web && npm run build && cd .. && \
    CGO_ENABLED=0 go build -tags prod \
      -ldflags="-s -w -X main.VERSION=${VERSION} -X github.com/ChevalRouting/routier/pkg/server/api/app.Version=${VERSION}" \
      -o /out/routier ./cmd/routier && \
    CGO_ENABLED=0 go build -tags prod \
      -ldflags="-s -w -X main.VERSION=${VERSION} -X github.com/ChevalRouting/routier/pkg/server/api/app.Version=${VERSION}" \
      -o /out/routier-ui ./cmd/routier-ui


FROM alpine:3.23

# openrc pulls each daemon's *-openrc service script through Alpine install_if,
# so rc-service controls them exactly as on a real node.
RUN --mount=type=cache,target=/etc/apk/cache \
    apk add \
      openrc util-linux-openrc doas \
      nftables frr frr-openrc frr-pythontools wireguard-tools iproute2 dhcpcd cronie \
      openssl chrony keepalived conntrack-tools zstd rrdtool radvd \
      kea-dhcp4 kea-dhcp6 kea-dhcp-ddns kea-hook-lease-cmds bind bind-tools \
      mtr traceroute curl nmap

COPY --from=build /out/routier /usr/bin/routier
COPY --from=build /out/routier-ui /usr/bin/routier-ui
COPY scripts/routier-net.initd /etc/init.d/routier-net
COPY scripts/routier.initd /etc/init.d/routier
COPY scripts/routier-ui.initd /etc/init.d/routier-ui
COPY scripts/routier-firstboot.initd /etc/init.d/routier-firstboot
COPY scripts/routier.doas /etc/doas.d/routier.conf
COPY scripts/routier.crontab /etc/cron.d/routier
COPY scripts/routier.profile /etc/profile.d/routier.sh
COPY dev/entrypoint.sh /entrypoint.sh

RUN adduser -D -s /bin/ash -h /home/routier routier && \
    printf 'routier:routier\n' | chpasswd && \
    addgroup routier wheel && addgroup routier root && \
    chgrp wheel /usr/bin/routier && chmod 4750 /usr/bin/routier && \
    chmod 0755 /usr/bin/routier-ui /entrypoint.sh \
      /etc/init.d/routier-net /etc/init.d/routier /etc/init.d/routier-ui /etc/init.d/routier-firstboot && \
    install -dm750 /etc/routier /etc/routier/templates && \
    install -dm755 /etc/nftables.d && \
    install -dm700 /var/lib/routier /var/lib/routier/snapshots && \
    printf 'routier\n' > /var/lib/routier/ui-seed-password && chmod 600 /var/lib/routier/ui-seed-password && \
    (umask 077 && openssl ecparam -genkey -name prime256v1 -noout -out /var/lib/routier/tls.key) && \
    openssl req -new -x509 -key /var/lib/routier/tls.key -out /var/lib/routier/tls.crt \
      -days 3650 -subj "/CN=routier" && \
    (/usr/bin/routier version > /etc/routier/version 2>/dev/null || true) && \
    rc-update add routier-net boot && \
    rc-update add routier default && \
    rc-update add routier-ui default && \
    rc-update add cronie default

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
