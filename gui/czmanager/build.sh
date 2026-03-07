#!/bin/bash

echo "========================================"
echo "  CZ Agent GUI - Linux Build"
echo "========================================"
echo ""

cd "$(dirname "$0")"

echo "[1/2] Building Linux AMD64..."
go run github.com/wailsapp/wails/v2/cmd/wails@latest build -platform linux/amd64
if [ $? -ne 0 ]; then
    echo "BUILD FAILED!"
    exit 1
fi
cp build/bin/cz-agent-gui build/bin/cz-agent-gui-linux-amd64
echo "      OK!"

echo "[2/2] Building Linux ARM64..."
go run github.com/wailsapp/wails/v2/cmd/wails@latest build -platform linux/arm64
if [ $? -ne 0 ]; then
    echo "BUILD FAILED!"
    exit 1
fi
cp build/bin/cz-agent-gui build/bin/cz-agent-gui-linux-arm64
echo "      OK!"

echo ""
echo "========================================"
echo "  Build complete!"
echo "========================================"
ls -la build/bin/
