#!/bin/bash
set -e
cd "$(dirname "$0")/.."

echo "==> Building frontend..."
cd frontend
npm install --prefer-offline
npm run build
cd ..

echo "==> Cross-compiling Linux amd64 binary..."
mkdir -p build
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/goproxy-webserver ./cmd/webserver/

echo "==> Preparing deployment directory..."
DEPLOY_DIR="build/goproxy-linux-amd64"
rm -rf "$DEPLOY_DIR"
mkdir -p "$DEPLOY_DIR/frontend/dist"
cp build/goproxy-webserver "$DEPLOY_DIR/"
cp -r frontend/dist/* "$DEPLOY_DIR/frontend/dist/"

echo "==> Done: $DEPLOY_DIR/"
echo "    Binary: $DEPLOY_DIR/goproxy-webserver"
echo "    Static: $DEPLOY_DIR/frontend/dist/"
echo ""
echo "Deploy to Linux:"
echo "  1. Copy $DEPLOY_DIR/ to target server"
echo "  2. Create config: ./goproxy-webserver -write-default"
echo "  3. Start server: ./goproxy-webserver"
echo "  4. Open browser: http://<server-ip>:9090"
echo "  5. Default login: admin / admin"
