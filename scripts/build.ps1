$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path bin | Out-Null
$version = "0.1.0"
$commit = "unknown"
try { $commit = (git rev-parse --short HEAD).Trim() } catch {}
go build -ldflags "-X github.com/TEHILLA-O/fork_ledger/internal/version.Version=$version -X github.com/TEHILLA-O/fork_ledger/internal/version.Commit=$commit" -o bin/forkledger.exe ./cmd/forkledger
Write-Host "wrote bin/forkledger.exe"
