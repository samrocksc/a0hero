#!/usr/bin/env bash
# Build helper for a0hero — injects version info via ldflags.
# Usage: ./build.sh [version] [commit]
# Called by Makefile and GitHub Actions.

set -euo pipefail

VERSION="${1:-dev}"
COMMIT="${2:-$(git rev-parse --short HEAD 2>/dev/null || echo 'none')}"
BUILDDATE="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
LDFLAGS="-s -w \
  -X github.com/samrocksc/a0hero/version.Version=${VERSION} \
  -X github.com/samrocksc/a0hero/version.Commit=${COMMIT} \
  -X github.com/samrocksc/a0hero/version.BuildDate=${BUILDDATE}"

echo "Building a0hero ${VERSION} (commit: ${COMMIT}, built: ${BUILDDATE})"

eval go build -ldflags "'${LDFLAGS}'" -o "${OUTPUT:-bin/a0hero}" ./cmd/a0hero/

echo "OK → ${OUTPUT:-bin/a0hero}"