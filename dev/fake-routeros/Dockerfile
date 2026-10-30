# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    if [ -d vendor ]; then export GOFLAGS=-mod=vendor; else go mod download; fi && \
    CGO_ENABLED=0 go build -o /fake-routeros ./dev/fake-routeros

FROM alpine:3.23

COPY --from=build /fake-routeros /usr/local/bin/fake-routeros
COPY dev/fake-routeros/crs310.json /etc/fake-routeros/crs310.json
EXPOSE 443
ENTRYPOINT ["fake-routeros", "-data", "/etc/fake-routeros/crs310.json"]
