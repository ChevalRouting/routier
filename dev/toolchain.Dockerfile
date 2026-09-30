# syntax=docker/dockerfile:1

FROM golang:1.25-alpine

ARG SQLC_VERSION=v1.30.0
ARG GOLANGCI_LINT_VERSION=v2.12.2
ARG SWAG_VERSION=v2.0.0-rc5
ARG OAPI_CODEGEN_VERSION=v2.1.0
ARG OPENAPI_GENERATOR_VERSION=7.13.0
ARG TARGETARCH

RUN apk add --no-cache build-base ca-certificates curl git make nodejs npm openjdk21-jre su-exec zstd
RUN sqlc_version="${SQLC_VERSION#v}" \
    && curl -fsSL "https://github.com/sqlc-dev/sqlc/releases/download/${SQLC_VERSION}/sqlc_${sqlc_version}_linux_${TARGETARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin sqlc
RUN lint_version="${GOLANGCI_LINT_VERSION#v}" \
    && curl -fsSL "https://github.com/golangci/golangci-lint/releases/download/${GOLANGCI_LINT_VERSION}/golangci-lint-${lint_version}-linux-${TARGETARCH}.tar.gz" \
    | tar -xz --strip-components=1 -C /usr/local/bin "golangci-lint-${lint_version}-linux-${TARGETARCH}/golangci-lint"
RUN GOBIN=/usr/local/bin go install "github.com/swaggo/swag/v2/cmd/swag@${SWAG_VERSION}" \
    && GOBIN=/usr/local/bin go install "github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION}"
RUN curl -fsSL "https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/${OPENAPI_GENERATOR_VERSION}/openapi-generator-cli-${OPENAPI_GENERATOR_VERSION}.jar" \
    -o /usr/local/lib/openapi-generator-cli.jar

COPY dev/toolchain-entrypoint.sh /usr/local/bin/toolchain-entrypoint
RUN chmod 0755 /usr/local/bin/toolchain-entrypoint

WORKDIR /workspace
ENTRYPOINT ["toolchain-entrypoint"]
