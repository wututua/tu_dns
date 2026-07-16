#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

cd "$ROOT_DIR/frontend"
npm ci
npm run build

cd "$ROOT_DIR"
mkdir -p bin
go build -trimpath -o bin/tudns .
