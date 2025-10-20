#!/bin/bash

set -e

echo "Starting Aegis Gateway..."
echo ""

if [ ! -f "go.mod" ]; then
  echo "Error: go.mod not found. Run from project root."
  exit 1
fi

echo "Installing dependencies..."
go mod tidy > /dev/null 2>&1

echo "Building binary..."
go build -o bin/aegis-gateway cmd/aegis/main.go

echo ""
echo "Starting gateway..."
echo "================================"
./bin/aegis-gateway
