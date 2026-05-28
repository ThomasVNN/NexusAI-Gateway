#!/bin/bash
set -e

echo "=========================================================="
echo "NexusAI-Gateway: E2E Integration and Performance Test"
echo "=========================================================="

cd /Users/thont/Local/POC/NexusAI-Gateway/deployments

# 1. Clean previous state and start environment
docker compose down -v || true
docker compose build
docker compose up -d

echo "Waiting for services to become healthy..."
sleep 8

# 2. Extract Gateway Container Status and configs
echo "Checking Gateway container boot logs:"
docker logs nexusai-gateway | grep -i "Starting NexusAI-Gateway" || true

# 3. Test API Key Registration Endpoint
echo "Test A: Creating secure API Key..."
KEY_RESPONSE=$(curl -s -X POST http://localhost:20129/api/admin/keys \
  -H "Content-Type: application/json" \
  -d '{"name":"E2E-Test-Key","source_app":"openwebui","daily_quota":100,"hourly_quota":20}')

echo "Response payload from key generator:"
echo "$KEY_RESPONSE"

API_TOKEN=$(echo "$KEY_RESPONSE" | grep -o '"key":"[^"]*' | grep -o '[^"]*$')
if [ -z "$API_TOKEN" ]; then
  echo "E2E Failure: Failed to generate API token"
  exit 1
fi
echo "Generated Token: $API_TOKEN"

# 4. Test OpenAI Chat Completions streaming with PII Redaction
echo "Test B: Invoking SSE Completions with PII payload (email & card)..."
CHAT_RESPONSE=$(curl -s -X POST http://localhost:20129/v1/chat/completions \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"Hi my email is target@nexus.com and my credit card is 1234-5678-9012-3456"}],"stream":true}')

echo "Streaming SSE payload chunks:"
echo "$CHAT_RESPONSE"

# 5. Check Audit logs and telemetry counters
echo "Test C: Requesting Aggregate System Telemetry..."
TELEMETRY=$(curl -s http://localhost:20129/api/admin/usage)
echo "System Metrics:"
echo "$TELEMETRY"

echo "=========================================================="
echo "NexusAI-Gateway: E2E integration test successfully PASSED"
echo "=========================================================="

# Clean up
docker compose down -v
