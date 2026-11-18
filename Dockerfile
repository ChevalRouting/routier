FROM alpine:3.23

COPY APKBUILD /tmp/APKBUILD

RUN apk update && \
  apk add alpine-sdk make go nodejs npm s6 icu-data-full frr-openrc kea \
  alpine-conf xorriso squashfs-tools mtools dosfstools grub git && \
  apk add $(sed -n 's/^depends="\(.*\)"$/\1/p' /tmp/APKBUILD) && \
  rm -f /tmp/APKBUILD && \
  adduser -D builder && \
  addgroup builder abuild

RUN mkdir /work && chown builder:builder /work

USER builder
WORKDIR /home/builder
COPY --chown=builder:builder go.mod go.sum /home/builder/

ENV GOPRIVATE=github.com/m-vinc/*

RUN go mod download

USER root
