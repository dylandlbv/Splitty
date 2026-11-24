#!/usr/bin/env bash
# Local test pipeline, mirroring the CI "Tests" workflow (.github/workflows/test.yml):
# backend lint (gofmt, go vet), backend unit + API + e2e tests with -race,
# then frontend install + typecheck + build.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> backend: gofmt"
cd backend
test -z "$(gofmt -l .)"

echo "==> backend: go vet"
go vet ./...

echo "==> backend: go test ./... -race (unit + API scenarios + e2e smoke with real binary)"
go test ./... -race -count=1
cd ..

if command -v npm >/dev/null 2>&1; then
  echo "==> frontend: npm ci + typecheck + build"
  cd frontend
  npm ci
  npm run build
  cd ..
else
  echo "==> frontend: npm not found, skipping frontend build"
fi

echo "==> all checks passed"
