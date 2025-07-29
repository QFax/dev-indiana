#!/bin/bash

# This script performs a basic functionality test of the Gemini API proxy.
# 1. It checks the health of the proxy.
# 2. It sends a sample request to the Gemini API through the proxy.

# --- Configuration ---
PROXY_URL="http://localhost:8080"

# Load environment variables from .env file.
# This assumes the script is run from the root of the project where the .env file is.
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

# --- Health Check ---
echo "--- Running Health Check ---"
echo "Pinging proxy at $PROXY_URL/health"

HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$PROXY_URL/health")

if [ "$HEALTH_STATUS" -eq 200 ]; then
  echo "✅ Health check successful (Status: $HEALTH_STATUS)"
else
  echo "❌ Health check failed (Status: $HEALTH_STATUS)"
  echo "Please ensure the proxy server is running."
  exit 1
fi

# --- Gemini API Basic Test ---
echo ""
echo "--- Running Gemini API Basic Test ---"

if [ -z "$PROXY_API_KEY" ]; then
  echo "❌ PROXY_API_KEY is not set in the .env file."
  exit 1
fi

JSON_PAYLOAD='{
  "contents": [
    {
      "parts": [
        {
          "text": "Why is the sky blue?"
        }
      ]
    }
  ]
}'

# This assumes the proxy forwards requests to a path like /v1beta/models/gemini-2.5-flash-lite:generateContent
# The model can be changed here if needed.
API_ENDPOINT="$PROXY_URL/v1beta/models/gemini-2.5-flash-lite:generateContent"
RESPONSE_FILE=$(mktemp)

echo "Sending POST request to $API_ENDPOINT"

RESPONSE_CODE=$(curl -s -o "$RESPONSE_FILE" -w "%{http_code}" \
  -X POST \
  -H "Content-Type: application/json" \
  -H "X-Proxy-API-Key: $PROXY_API_KEY" \
  -d "$JSON_PAYLOAD" \
  "$API_ENDPOINT")

echo ""
if [ "$RESPONSE_CODE" -eq 200 ]; then
  echo "✅ Gemini API request successful (Status: $RESPONSE_CODE)"
  echo "--- Proxy Response ---"
  cat "$RESPONSE_FILE"
  echo ""
  echo "----------------------"
else
  echo "❌ Gemini API request failed (Status: $RESPONSE_CODE)"
  echo "--- Proxy Response ---"
  cat "$RESPONSE_FILE"
  echo ""
  echo "----------------------"
  rm "$RESPONSE_FILE"
  exit 1
fi

rm "$RESPONSE_FILE"

echo ""
echo "✅ All tests passed successfully."