#!/usr/bin/env bash
# Builds PomoPet.exe (Windows x64) from any OS with Go installed.
set -euo pipefail
cd "$(dirname "$0")"
go run ./cmd/genicon                                  # winres/icon.png
go run github.com/tc-hib/go-winres@v0.3.3 make --arch amd64   # rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o PomoPet.exe .
echo "Built $(pwd)/PomoPet.exe"
