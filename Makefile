VERSION := 0.0.0
RELEASE := 0

ARCH_RAW := $(shell uname -m)
ARCH     := $(if $(filter arm64,$(ARCH_RAW)),aarch64,$(ARCH_RAW))
APKS = dist/$(ARCH)/routier-$(VERSION)-r$(RELEASE).apk dist/$(ARCH)/routier-openrc-$(VERSION)-r$(RELEASE).apk

EMBED ?= 0
LOCAL_ANYK ?= 0

ifeq ($(LOCAL_ANYK),1)
APK_ANYK_MOUNT := -v $(CURDIR)/../anyk:/anyk:z
else
APK_ANYK_MOUNT :=
endif

ifeq ($(EMBED),1)
GO_TAGS   := -tags prod
WEB_BUILD := web-build
else
GO_TAGS   :=
WEB_BUILD :=
endif

BINS := $(notdir $(wildcard cmd/*))
WEB_SRCS := $(shell find web/src web/public -type f 2>/dev/null) web/package.json web/vite.config.ts

DOCKER_STAMP := .docker-image-$(VERSION)-$(RELEASE)

PODMAN_TTY ?= -it

ALPINE_VERSION := 3.23
ISO_APORTS     ?= ./dist/aports
ISO_APORTS_MOUNT := -v $(abspath $(ISO_APORTS)):/aports:z

REPO_DIR := dist/$(ARCH)
WWW_DIR  ?= /var/lib/Routier
REPO_URL ?= http://localhost/$(ARCH)/

.PHONY: docs tools website website-serve website-version website-version-dev
.PHONY: build install clean test lint apk apk-index www builder iso run-iso run-disk run-kernel web-build $(addprefix build.,$(BINS))

build: $(addprefix build.,$(BINS))

$(addprefix build.,$(BINS)): build.%: $(WEB_BUILD)
	CGO_ENABLED=0 go build $(GO_TAGS) -ldflags="-s -w -X main.VERSION=$(VERSION) -X github.com/ChevalRouting/routier/pkg/api/app.Version=$(VERSION)" -o $* ./cmd/$*

web-build:
	cd web && npm install && npm run build

GOBIN := $(shell go env GOPATH)/bin

tools:
	go install github.com/swaggo/swag/v2/cmd/swag@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

docs:
	dirs="./pkg/api"; for d in pkg/api/routes/*/; do dirs="$$dirs,./$${d%/}"; done; \
	$(GOBIN)/swag init --v3.1 -g doc.go -d "$$dirs" --parseDependency --parseInternal --requiredByDefault -o pkg/api/docs
	$(GOBIN)/oapi-codegen -generate types,client -package routierclient -o pkg/client/generated/routier.gen.go pkg/api/docs/swagger.json
	npx --yes @openapitools/openapi-generator-cli generate -i /local/pkg/api/docs/swagger.json -g typescript-fetch -o /local/web/src/api \
		--additional-properties=modelPropertyNaming=original,paramNaming=original,supportsES6=true \
		--reserved-words-mappings interface=interface,in=in,new=new
	cd web && npm install --no-save swagger-ui-dist >/dev/null 2>&1 || npm install --no-save swagger-ui-dist
	mkdir -p pkg/api/docs/swagger-ui
	cp web/node_modules/swagger-ui-dist/swagger-ui.css web/node_modules/swagger-ui-dist/swagger-ui-bundle.js pkg/api/docs/swagger-ui/

website:
	cd website && npm install && npm run build

website-serve:
	cd website && npm install && npm start

website-version:
	@test "$(VERSION)" != "0.0.0" || { echo "set VERSION=x.y (e.g. VERSION=0.5)"; exit 1; }
	cd website && npm install && npm run docusaurus docs:version $(VERSION)

website-version-dev:
	cd website && npm install && sh scripts/version-dev.sh

install: build
	$(foreach bin,$(BINS),install -m 755 $(bin) /usr/local/bin/$(bin);)

clean:
	rm -rf $(BINS) pkg/api/dist web/node_modules $(DOCKER_STAMP)

test:
	go vet ./...
	go test ./...

lint:
	golangci-lint run ./...
	cd web && npx eslint src

$(DOCKER_STAMP): Dockerfile
	podman build -t "routier:${VERSION}-${RELEASE}" -f Dockerfile .
	touch $@

builder: $(DOCKER_STAMP)

routier.rsa:
	openssl genrsa -out routier.rsa 4096
	openssl rsa -in routier.rsa -pubout -out routier.rsa.pub
	chmod 600 routier.rsa

routier.rsa.pub: routier.rsa

$(word 1,$(APKS)): $(DOCKER_STAMP) routier.rsa routier.rsa.pub
	EMBED=1 LOCAL_ANYK=$(LOCAL_ANYK) podman run $(PODMAN_TTY) \
		-e RELEASE="$(RELEASE)" -e VERSION="$(VERSION)" -e LOCAL_ANYK="$(LOCAL_ANYK)" \
		-v $(CURDIR):/app:z $(APK_ANYK_MOUNT) \
		--entrypoint sh "routier:${VERSION}-${RELEASE}" /app/scripts/mkpkg.sh

$(word 2,$(APKS)): $(word 1,$(APKS))

apk: $(APKS)

apk-index: $(APKS)
	podman run --rm -v $(CURDIR):/app:z \
		--entrypoint sh "routier:${VERSION}-${RELEASE}" /app/scripts/mkindex.sh

www: apk-index
	rm -rf $(WWW_DIR)/*
	mkdir -p $(WWW_DIR)/$(ARCH)
	cp $(REPO_DIR)/*.apk $(REPO_DIR)/APKINDEX.tar.gz $(WWW_DIR)/$(ARCH)/
	cp routier.rsa.pub $(WWW_DIR)/
	@echo "apk repo assembled in $(WWW_DIR)/$(ARCH) (serve as $(REPO_URL))"

apk_debug: $(DOCKER_STAMP)
	EMBED=1 LOCAL_ANYK=$(LOCAL_ANYK) podman run -it -e RELEASE="$(RELEASE)" -e VERSION="$(VERSION)" -e LOCAL_ANYK="$(LOCAL_ANYK)" -v $(CURDIR):/app:z $(APK_ANYK_MOUNT) --entrypoint ash "routier:${VERSION}-${RELEASE}"

run-iso:
	sh scripts/run-iso.sh

run-disk:
	sh scripts/run-iso.sh ""

run-kernel:
	sh scripts/run-kernel.sh

iso: apk
	@[ -d "$(ISO_APORTS)/.git" ] || git clone --depth=1 \
		--branch $(ALPINE_VERSION)-stable \
		https://gitlab.alpinelinux.org/alpine/aports "$(ISO_APORTS)"
	podman run $(PODMAN_TTY) --rm --privileged \
		-e VERSION="$(VERSION)" \
		-e ROUTIER_REPO="/app/dist" \
		-e APORTS="/aports" \
		-e OUTDIR="/app/dist/iso" \
		-v $(CURDIR):/app:z \
		$(ISO_APORTS_MOUNT) \
		"routier:${VERSION}-${RELEASE}" \
		sh /app/scripts/build-iso.sh

