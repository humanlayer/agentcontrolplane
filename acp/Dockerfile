# Build the manager binary
FROM docker.io/golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/go/pkg/mod

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum


# Cache the go modules and build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/

# Build with caching
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Install uv/uvx
FROM debian:bookworm-slim AS uv-installer
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates
ADD https://astral.sh/uv/install.sh /uv-installer.sh
RUN sh /uv-installer.sh && rm /uv-installer.sh

# Python slim image provides both Python and a minimal Debian
FROM python:3.12-slim-bookworm
WORKDIR /

# Install Node.js and NPM/NPX
RUN apt-get update && apt-get install -y --no-install-recommends \
    nodejs npm \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Copy our manager binary from the builder stage
COPY --from=builder /workspace/manager .

# Copy uv/uvx from the installer stage
COPY --from=uv-installer /root/.local/bin/uv /usr/local/bin/uv
COPY --from=uv-installer /root/.local/bin/uvx /usr/local/bin/uvx

# Create non-root user to match the 65532 UID from distroless
RUN groupadd -g 65532 nonroot && \
    useradd -u 65532 -g nonroot -s /bin/bash -m nonroot

USER 65532:65532

ENTRYPOINT ["/manager"]
