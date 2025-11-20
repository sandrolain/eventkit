#!/bin/bash
# Example script to test httptool multipart file upload
# This script demonstrates various use cases of the multipart upload feature

set -e

echo "=== HTTPTool Multipart Upload Examples ==="
echo ""

# Create test files
mkdir -p /tmp/httptool-test
echo "This is a test document" > /tmp/httptool-test/document.txt
echo '{"test": "data"}' > /tmp/httptool-test/data.json
echo "Binary content" > /tmp/httptool-test/binary.dat

echo "Test files created in /tmp/httptool-test/"
echo ""

# Start a simple HTTP server in the background to receive uploads
echo "Starting HTTP server on :9090..."
httptool serve --address :9090 &
SERVER_PID=$!
sleep 2

echo ""
echo "=== Example 1: Single file upload ==="
httptool send \
  --address http://localhost:9090 \
  --path /upload \
  --file document=/tmp/httptool-test/document.txt \
  --interval 0s

echo ""
echo "=== Example 2: Multiple files upload ==="
httptool send \
  --address http://localhost:9090 \
  --path /upload \
  --file document=/tmp/httptool-test/document.txt \
  --file data=/tmp/httptool-test/data.json \
  --interval 0s

echo ""
echo "=== Example 3: Files with form fields ==="
httptool send \
  --address http://localhost:9090 \
  --path /upload \
  --file document=/tmp/httptool-test/document.txt \
  --form-field username=testuser \
  --form-field email=test@example.com \
  --form-field description="Test upload from httptool" \
  --interval 0s

echo ""
echo "=== Example 4: Form fields with templates ==="
httptool send \
  --address http://localhost:9090 \
  --path /upload \
  --file document=/tmp/httptool-test/document.txt \
  --form-field timestamp={{nowtime}} \
  --form-field upload_id={{uuid}} \
  --form-field counter={{counter}} \
  --interval 0s

echo ""
echo "=== Example 5: Custom template delimiters ==="
httptool send \
  --address http://localhost:9090 \
  --path /upload \
  --template-open '<!' \
  --template-close '!>' \
  --file document=/tmp/httptool-test/document.txt \
  --form-field timestamp=<!nowtime!> \
  --interval 0s

# Cleanup
echo ""
echo "Stopping HTTP server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true

echo ""
echo "Cleaning up test files..."
rm -rf /tmp/httptool-test

echo ""
echo "=== All examples completed successfully ==="
