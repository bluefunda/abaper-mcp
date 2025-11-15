#!/bin/bash

# Simple test script for the MCP server
# This simulates what Claude Desktop would send

echo "Testing ABAPER MCP Server..."
echo ""

# Set dummy environment variables for testing
export SAP_HOST="https://test.example.com:8000"
export SAP_CLIENT="100"
export SAP_USERNAME="testuser"
export SAP_PASSWORD="testpass"

# Send initialize message and capture response
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | \
  ./abaper-mcp 2>&1 | head -50 &

PID=$!
sleep 2
kill $PID 2>/dev/null

echo ""
echo "If you see JSON-RPC responses above without panics, the server is working correctly!"
