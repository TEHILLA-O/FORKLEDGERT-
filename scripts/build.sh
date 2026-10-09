#!/usr/bin/env bash
set -euo pipefail
mkdir -p bin
VERSION="${VERSION:-0.1.0}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
go build -ldflags "-X github.com/TEHILLA-O/fork_ledger/internal/version.Version=${VERSION} -X github.com/TEHILLA-O/fork_ledger/internal/version.Commit=${COMMIT}" -o bin/forkledger ./cmd/forkledger
echo "wrote bin/forkledger"
