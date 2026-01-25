#!/bin/bash
# Script to set or update device location using the Logger API
# Usage: ./set_location.sh <source> <location> [server_url] [port]
# Examples:
#   ./set_location.sh sensor-01 "Building A, Room 101"
#   ./set_location.sh sensor-01 "Building A, Room 101" localhost
#   ./set_location.sh sensor-01 "Building A, Room 101" 192.168.1.100 8765
#   ./set_location.sh sensor-01 "Building A, Room 101" http://192.168.1.100:8765

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEFAULT_PORT=8765
DEFAULT_HOST="localhost"

# Check arguments
if [ $# -lt 2 ]; then
    echo -e "${RED}Error: Missing required arguments${NC}"
    echo ""
    echo "Usage: $0 <source> <location> [server_url] [port]"
    echo ""
    echo "Arguments:"
    echo "  source      - Device identifier (e.g., sensor-01, arduino-1)"
    echo "  location    - Location string (e.g., 'Building A, Room 101')"
    echo "  server_url  - Server hostname or IP (default: localhost)"
    echo "  port        - Server port (default: 8765)"
    echo ""
    echo "Examples:"
    echo "  $0 sensor-01 \"Building A, Room 101\""
    echo "  $0 sensor-01 \"Building A, Room 101\" 192.168.1.100"
    echo "  $0 sensor-01 \"Building A, Room 101\" 192.168.1.100 8080"
    echo "  $0 sensor-01 \"Building A, Room 101\" http://192.168.1.100:8765"
    echo ""
    exit 1
fi

SOURCE="$1"
LOCATION="$2"
PARAM3="${3:-$DEFAULT_HOST}"
PARAM4="${4:-$DEFAULT_PORT}"

# Determine API_URL from parameters
if [[ "$PARAM3" == http://* ]] || [[ "$PARAM3" == https://* ]]; then
    # Full URL provided, use it as-is
    API_URL="$PARAM3"
elif [[ "$PARAM3" == localhost ]] || [[ "$PARAM3" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
    # IP address or localhost provided, construct URL
    API_URL="http://${PARAM3}:${PARAM4}"
else
    # Assume it's a hostname or use as-is
    API_URL="http://${PARAM3}:${PARAM4}"
fi

LOGGER_ENDPOINT="${API_URL}/api/logger"

# Get current datetime in ISO format
DATETIME=$(date -u +"%Y-%m-%dT%H:%M:%S")

echo -e "${BLUE}Setting location for device: ${YELLOW}${SOURCE}${NC}"
echo -e "${BLUE}Location: ${YELLOW}${LOCATION}${NC}"
echo -e "${BLUE}Server: ${YELLOW}${API_URL}${NC}"
echo ""

# Check if server is running
if ! curl -s -f "${API_URL}/health" > /dev/null 2>&1; then
    echo -e "${RED}Error: Server is not responding at ${API_URL}${NC}"
    echo "Please ensure the logger server is running."
    exit 1
fi

# Make the API call
RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d "{
        \"datetime\": \"${DATETIME}\",
        \"transaction\": \"set_location\",
        \"name\": \"location\",
        \"value\": \"${LOCATION}\",
        \"source\": \"${SOURCE}\"
    }")

# Check if the response indicates success
if echo "$RESPONSE" | grep -q "\"success\".*true"; then
    ENTRY_ID=$(echo "$RESPONSE" | grep -o '"id"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' || echo "unknown")
    echo -e "${GREEN}✓ Success${NC}: Location set successfully"
    echo -e "  Entry ID: ${ENTRY_ID}"
    echo ""
    echo "All future log entries from '${SOURCE}' will include this location."
    exit 0
else
    echo -e "${RED}✗ Error${NC}: Failed to set location"
    echo ""
    echo "Response:"
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
    exit 1
fi
