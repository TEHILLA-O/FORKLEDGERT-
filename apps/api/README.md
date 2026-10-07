# apps/api

The hosted API is intentionally not a v0.1 deliverable.

The CLI writes a JSON report. That report is the contract. When organisation-wide discovery exists, this service will ingest those reports into PostgreSQL using `internal/storage/migrations`.

Until the analyzer is boringly reliable, this directory stays a placeholder so the monorepo layout matches the product direction without shipping an empty HTTP surface.
