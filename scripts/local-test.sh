#!/bin/bash
# Local testing script for abaper-mcp with cai-llm-router
#
# Prerequisites:
# - NATS CLI installed and configured with 'test' context
# - cai-llm-router running and connected to NATS
# - abaper-mcp running in SSE mode on port 8015
#
# Usage:
#   ./scripts/local-test.sh [prompt]
#
# Examples:
#   ./scripts/local-test.sh "List available tools"
#   ./scripts/local-test.sh "Get program ZTEST"

set -e

# Configuration
NATS_CONTEXT="${NATS_CONTEXT:-test}"
MCP_SERVER_NAME="${MCP_SERVER_NAME:-abaper-mcp}"
MCP_SERVER_URL="${MCP_SERVER_URL:-http://localhost:8015/sse}"
MODEL="${MODEL:-groq}"
CHAT_ID="mcp-$(date +%s)"
PROMPT="${1:-List available tools}"

echo "=== abaper-mcp Local Test ==="
echo "Chat ID:    $CHAT_ID"
echo "Model:      $MODEL"
echo "MCP Server: $MCP_SERVER_NAME ($MCP_SERVER_URL)"
echo "Prompt:     $PROMPT"
echo "============================"
echo ""

# Check if abaper-mcp is running
if ! curl -s http://localhost:8015/health > /dev/null 2>&1; then
    echo "ERROR: abaper-mcp is not running on port 8015"
    echo ""
    echo "Start it with:"
    echo "  cd ~/src/abaper-mcp && ABAPER_MODE=sse LOG_FORMAT=console LOG_LEVEL=debug ./abaper-mcp"
    exit 1
fi

echo "abaper-mcp health check: OK"
echo ""

# Subscribe to responses in background
echo "Subscribing to responses..."
nats sub "test.realm.user.chat.${CHAT_ID}.>" --context "$NATS_CONTEXT" --count 50 &
SUB_PID=$!
sleep 1

# Publish the request
echo ""
echo "Publishing request..."
nats pub --context "$NATS_CONTEXT" \
    --reply "test.realm.user.chat.${CHAT_ID}.response" \
    "test.realm.user.chat.${CHAT_ID}" \
    "{\"type\":\"Human\",\"model\":\"${MODEL}\",\"prompt\":\"${PROMPT}\",\"mcp_server_name\":\"${MCP_SERVER_NAME}\",\"mcp_server_url\":\"${MCP_SERVER_URL}\"}"

echo ""
echo "Waiting for responses (30s timeout)..."

# Wait for subscriber or timeout
sleep 30
kill $SUB_PID 2>/dev/null || true

echo ""
echo "=== Test Complete ==="
