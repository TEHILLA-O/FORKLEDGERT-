.PHONY: build test vet fixtures web

VERSION ?= 0.1.0
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

build:
	go build -ldflags "-X github.com/TEHILLA-O/fork_ledger/internal/version.Version=$(VERSION) -X github.com/TEHILLA-O/fork_ledger/internal/version.Commit=$(COMMIT)" -o bin/forkledger ./cmd/forkledger

test:
	go test ./...

vet:
	go vet ./...

fixtures:
	go test ./internal/scanner -count=1 -run TestAnalyzeForkWorld -v

web:
	cd apps/web && npm install && npm run build
