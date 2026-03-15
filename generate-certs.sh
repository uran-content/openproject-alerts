#!/bin/sh
set -e

CERTS_DIR="./certs"
mkdir -p "$CERTS_DIR"

if [ -f "$CERTS_DIR/server.crt" ] && [ -f "$CERTS_DIR/server.key" ]; then
    echo "Certificates already exist in $CERTS_DIR, skipping generation."
    echo "Delete them and re-run this script to regenerate."
    exit 0
fi

echo "Generating self-signed certificate..."
openssl req -x509 -nodes -newkey rsa:4096 \
    -keyout "$CERTS_DIR/server.key" \
    -out "$CERTS_DIR/server.crt" \
    -days 365 \
    -subj "/CN=openproject-webhooks-bot"

echo "Certificates generated in $CERTS_DIR/"
echo "  - server.crt"
echo "  - server.key"
