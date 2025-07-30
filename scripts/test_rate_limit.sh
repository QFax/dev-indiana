#!/bin/bash

# This script tests if the API rate limit resets on a fixed minute boundary
# for a specific timezone (UTC or Pacific Time). It sends requests just
# before and just after the minute change to observe the behavior.

# --- Configuration & Argument Validation ---
set -e # Exit immediately if a command exits with a non-zero status.

MODE=$1
if [[ "$MODE" != "utc" && "$MODE" != "pacific" ]]; then
  echo "❌ Error: Invalid or missing mode."
  echo "Usage: $0 [utc|pacific]"
  exit 1
fi

PROXY_URL="http://localhost:8080"
REQUESTS_PER_BURST=3

# Load environment variables from .env file
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

if [ -z "$PROXY_API_KEY" ]; then
  echo "❌ PROXY_API_KEY is not set in the .env file."
  exit 1
fi

# --- Helper Functions ---

# Function to get the current second in the selected timezone
get_current_seconds() {
  if [ "$MODE" == "utc" ]; then
    date -u +%S
  else # pacific
    TZ="America/Los_Angeles" date +%S
  fi
}

# Function to get the current time string for logging
get_current_time_str() {
  if [ "$MODE" == "utc" ]; then
    date -u '+%H:%M:%S %Z'
  else # pacific
    TZ="America/Los_Angeles" date '+%H:%M:%S %Z'
  fi
}

# Function to send a burst of requests and update global counters
send_burst() {
  local burst_name=$1
  local success_count=0
  local rate_limited_count=0

  echo "--- Sending Burst '$burst_name' at $(get_current_time_str) ---"
  
  JSON_PAYLOAD='{"contents": [{"parts": [{"text": "Hi!"}]}]}'
  API_ENDPOINT="$PROXY_URL/v1beta/models/gemini-2.5-pro:generateContent"
  
  for i in $(seq 1 $REQUESTS_PER_BURST); do
    RESPONSE_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
      -X POST \
      -H "Content-Type: application/json" \
      -H "x-goog-api-Key: $PROXY_API_KEY" \
      -d "$JSON_PAYLOAD" \
      "$API_ENDPOINT")
    echo "Request #$i: Status $RESPONSE_CODE"
    
    if [ "$RESPONSE_CODE" -eq 200 ]; then
      success_count=$((success_count + 1))
    elif [ "$RESPONSE_CODE" -eq 429 ]; then
      rate_limited_count=$((rate_limited_count + 1))
    fi
  done
  
  # Update global counters
  TOTAL_SUCCESS=$((TOTAL_SUCCESS + success_count))
  TOTAL_RATE_LIMITED=$((TOTAL_RATE_LIMITED + rate_limited_count))
  
  echo "Burst '$burst_name' Summary: ✅ $success_count Successful, 🚦 $rate_limited_count Rate-Limited"
  echo "-----------------------------------"
}

# --- Test Execution ---

echo "🚀 Starting rate limit boundary test in '$MODE' mode."

# Global counters for the final summary
TOTAL_SUCCESS=0
TOTAL_RATE_LIMITED=0

# 1. Wait until 3 seconds before the next minute boundary.
# We target second 57 to start the first burst.
while true; do
  SECONDS_NOW=$(get_current_seconds)
  # remove leading zero if any, for bash arithmetic
  SECONDS_NOW=${SECONDS_NOW#0}
  
  if [ $SECONDS_NOW -ge 57 ]; then
    # If we are already past second 57, wait for the next minute.
    WAIT_TIME=$((60 - SECONDS_NOW + 57))
    echo "Current second ($SECONDS_NOW) is past the window. Waiting $WAIT_TIME seconds..."
    sleep $WAIT_TIME
  elif [ $SECONDS_NOW -lt 57 ]; then
    WAIT_TIME=$((57 - SECONDS_NOW))
    echo "Waiting $WAIT_TIME seconds to reach the minute boundary..."
    sleep $WAIT_TIME
  fi
  
  # Final check to ensure we are at second 57
  if [ $(get_current_seconds) -eq 57 ]; then
    break
  fi
done

# 2. Send the "Before" burst just before the minute flips.
send_burst "Before Boundary"

# 3. Send the "After" burst just after the minute flips.
# The previous burst takes ~2-3 seconds, so we are now across the boundary.
send_burst "After Boundary"

echo ""
echo "--- Final Summary ---"
echo "Mode: $MODE"
echo "Total Requests: $((REQUESTS_PER_BURST * 2))"
echo "✅ Total Successful: $TOTAL_SUCCESS"
echo "🚦 Total Rate-Limited: $TOTAL_RATE_LIMITED"
echo "✅ Test complete."
