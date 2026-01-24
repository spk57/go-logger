#!/bin/bash
# Test script for Logger API endpoints
# Usage: ./test/testlogger.sh [IP_ADDRESS_OR_URL] [PORT]
#   IP_ADDRESS_OR_URL: IP address (e.g., 192.168.1.100) or full URL (e.g., http://192.168.1.100:8765)
#   PORT: Port number (default: 8765, only used if first param is an IP address)
# Examples:
#   ./test/testlogger.sh                           # Uses http://localhost:8765
#   ./test/testlogger.sh 192.168.1.100             # Uses http://192.168.1.100:8765
#   ./test/testlogger.sh 192.168.1.100 8080        # Uses http://192.168.1.100:8080
#   ./test/testlogger.sh http://192.168.1.100:8765 # Uses the provided URL as-is

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEFAULT_PORT=8765
PARAM1="${1:-localhost}"
PARAM2="${2:-$DEFAULT_PORT}"

# Determine API_URL from parameters
if [[ "$PARAM1" == http://* ]] || [[ "$PARAM1" == https://* ]]; then
    # Full URL provided, use it as-is
    API_URL="$PARAM1"
elif [[ "$PARAM1" == localhost ]] || [[ "$PARAM1" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
    # IP address or localhost provided, construct URL
    API_URL="http://${PARAM1}:${PARAM2}"
else
    # Assume it's a hostname or use as-is
    API_URL="http://${PARAM1}:${PARAM2}"
fi

LOGGER_ENDPOINT="${API_URL}/api/logger"
STATS_ENDPOINT="${API_URL}/api/logger/stats"

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

print_test() {
    echo -e "${YELLOW}Test:${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

print_failure() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

check_response() {
    local response="$1"
    local expected_field="$2"
    local expected_value="$3"
    
    if echo "$response" | grep -q "\"$expected_field\""; then
        if [ -z "$expected_value" ]; then
            return 0
        elif echo "$response" | grep -q "$expected_value"; then
            return 0
        fi
    fi
    return 1
}

# Check if server is running
print_header "Checking Server Availability"
if curl -s -f "${API_URL}/health" > /dev/null 2>&1; then
    print_success "Server is running at ${API_URL}"
else
    print_failure "Server is not responding at ${API_URL}"
    echo "Please start the server with: ./start_server.sh"
    exit 1
fi

# Test 1: Create a log entry (POST /api/logger)
print_header "Test 1: Create Log Entry"
print_test "POST /api/logger - Valid entry"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T10:30:00",
        "transaction": "logging",
        "name": "temperature",
        "value": 23.5,
        "source": "sensor-01"
    }')

if check_response "$RESPONSE" "success" "true" && check_response "$RESPONSE" "id"; then
    ENTRY_ID=$(echo "$RESPONSE" | grep -o '"id"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*')
    print_success "Created log entry with ID: $ENTRY_ID"
    echo "Response: $RESPONSE"
else
    print_failure "Failed to create log entry"
    echo "Response: $RESPONSE"
fi

# Test 2: Create multiple log entries
print_header "Test 2: Create Multiple Log Entries"
for i in {1..3}; do
    print_test "Creating entry $i/3"
    RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{
            \"datetime\": \"2025-01-15T10:3${i}:00\",
            \"transaction\": \"logging\",
            \"name\": \"humidity\",
            \"value\": $((50 + i * 5)),
            \"source\": \"sensor-02\"
        }")
    
    if check_response "$RESPONSE" "success" "true"; then
        print_success "Entry $i created"
        echo "Response: $RESPONSE"
    else
        print_failure "Entry $i failed"
        echo "Response: $RESPONSE"
    fi
done

# Test 3: Get all log entries (GET /api/logger)
print_header "Test 3: Retrieve Log Entries"
print_test "GET /api/logger - All entries"

RESPONSE=$(curl -s "${LOGGER_ENDPOINT}")

if check_response "$RESPONSE" "success" "true" && check_response "$RESPONSE" "entries"; then
    TOTAL=$(echo "$RESPONSE" | grep -o '"total"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*')
    print_success "Retrieved entries. Total: $TOTAL"
    echo "Response: $RESPONSE"
else
    print_failure "Failed to retrieve entries"
    echo "Response: $RESPONSE"
fi

# Test 4: Get entries with pagination
print_header "Test 4: Pagination"
print_test "GET /api/logger?limit=2&offset=0"

RESPONSE=$(curl -s "${LOGGER_ENDPOINT}?limit=2&offset=0")

if check_response "$RESPONSE" "limit" "2" && check_response "$RESPONSE" "offset" "0"; then
    print_success "Pagination parameters working"
    echo "Response: $RESPONSE"
else
    print_failure "Pagination failed"
    echo "Response: $RESPONSE"
fi

# Test 5: Filter by source
print_header "Test 5: Filter by Source"
print_test "GET /api/logger?source=sensor-01"

RESPONSE=$(curl -s "${LOGGER_ENDPOINT}?source=sensor-01")

if check_response "$RESPONSE" "success" "true"; then
    print_success "Filter by source working"
    echo "Response: $RESPONSE"
    # Verify all entries have correct source
    if echo "$RESPONSE" | grep -q "sensor-01"; then
        print_success "All entries have correct source"
    fi
else
    print_failure "Filter by source failed"
    echo "Response: $RESPONSE"
fi

# Test 6: Filter by name
print_header "Test 6: Filter by Name"
print_test "GET /api/logger?name=temperature"

RESPONSE=$(curl -s "${LOGGER_ENDPOINT}?name=temperature")

if check_response "$RESPONSE" "success" "true"; then
    print_success "Filter by name working"
    echo "Response: $RESPONSE"
else
    print_failure "Filter by name failed"
    echo "Response: $RESPONSE"
fi

# Test 7: Combined filters
print_header "Test 7: Combined Filters"
print_test "GET /api/logger?source=sensor-02&name=humidity&limit=10"

RESPONSE=$(curl -s "${LOGGER_ENDPOINT}?source=sensor-02&name=humidity&limit=10")

if check_response "$RESPONSE" "success" "true"; then
    print_success "Combined filters working"
    echo "Response: $RESPONSE"
else
    print_failure "Combined filters failed"
    echo "Response: $RESPONSE"
fi

# Test 8: Get statistics (GET /api/logger/stats)
print_header "Test 8: Get Statistics"
print_test "GET /api/logger/stats"

RESPONSE=$(curl -s "${STATS_ENDPOINT}")

if check_response "$RESPONSE" "success" "true" && check_response "$RESPONSE" "total_entries"; then
    print_success "Statistics retrieved"
    echo "Response: $RESPONSE"
else
    print_failure "Failed to get statistics"
    echo "Response: $RESPONSE"
fi

# Test 9: Error handling - Missing required field
print_header "Test 9: Error Handling"
print_test "POST /api/logger - Missing datetime field"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "transaction": "logging",
        "name": "test",
        "value": 100,
        "source": "test"
    }')

if check_response "$RESPONSE" "success" "false" && echo "$RESPONSE" | grep -qi "missing\|datetime"; then
    print_success "Error handling working (missing datetime)"
    echo "Response: $RESPONSE"
else
    print_failure "Error handling failed"
    echo "Response: $RESPONSE"
fi

# Test 10: Error handling - Missing name field
print_test "POST /api/logger - Missing name field"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T10:30:00",
        "transaction": "logging",
        "value": 100,
        "source": "test"
    }')

if check_response "$RESPONSE" "success" "false" && echo "$RESPONSE" | grep -qi "missing.*name"; then
    print_success "Error handling working (missing name)"
    echo "Response: $RESPONSE"
else
    print_failure "Error handling failed"
    echo "Response: $RESPONSE"
fi

# Test 11: Error handling - Invalid datetime format
print_test "POST /api/logger - Invalid datetime format"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "invalid-date",
        "transaction": "logging",
        "name": "test",
        "value": 100,
        "source": "test"
    }')

if check_response "$RESPONSE" "success" "false" && echo "$RESPONSE" | grep -qi "datetime\|format\|invalid"; then
    print_success "Error handling working (invalid datetime)"
    echo "Response: $RESPONSE"
else
    print_failure "Error handling failed"
    echo "Response: $RESPONSE"
fi

# Test 12: Create entry with different value types
print_header "Test 12: Different Value Types"
print_test "POST /api/logger - String value"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T11:00:00",
        "transaction": "logging",
        "name": "status",
        "value": "online",
        "source": "system"
    }')

if check_response "$RESPONSE" "success" "true"; then
    print_success "String value accepted"
    echo "Response: $RESPONSE"
else
    print_failure "String value rejected"
    echo "Response: $RESPONSE"
fi

print_test "POST /api/logger - Integer value"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T11:01:00",
        "transaction": "logging",
        "name": "count",
        "value": 42,
        "source": "system"
    }')

if check_response "$RESPONSE" "success" "true"; then
    print_success "Integer value accepted"
    echo "Response: $RESPONSE"
else
    print_failure "Integer value rejected"
    echo "Response: $RESPONSE"
fi

# Test 13: Set location for a device
print_header "Test 13: Set Location Transaction"
print_test "POST /api/logger - Set location for sensor-03"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T11:10:00",
        "transaction": "set_location",
        "name": "location",
        "value": "Building A, Room 101",
        "source": "sensor-03"
    }')

if check_response "$RESPONSE" "success" "true" && check_response "$RESPONSE" "id"; then
    LOCATION_ID=$(echo "$RESPONSE" | grep -o '"id"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*')
    print_success "Location set with ID: $LOCATION_ID"
    echo "Response: $RESPONSE"
else
    print_failure "Failed to set location"
    echo "Response: $RESPONSE"
fi

# Test 14: Verify location is used in subsequent log entries
print_header "Test 14: Location Used in Log Entries"
print_test "POST /api/logger - Log entry after set_location (should use location as transaction)"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T11:15:00",
        "transaction": "logging",
        "name": "temperature",
        "value": 22.5,
        "source": "sensor-03"
    }')

if check_response "$RESPONSE" "success" "true"; then
    # Check if location is in the response
    if echo "$RESPONSE" | grep -q "location" && echo "$RESPONSE" | grep -q "Building A, Room 101"; then
        print_success "Location included in response"
    else
        print_failure "Location not found in response"
    fi
    # Check if original_transaction is present
    if echo "$RESPONSE" | grep -q "original_transaction"; then
        print_success "Original transaction preserved in response"
    else
        print_failure "Original transaction not found in response"
    fi
    echo "Response: $RESPONSE"
    
    # Verify the entry was stored with location as transaction
    ENTRY_ID=$(echo "$RESPONSE" | grep -o '"id"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*')
    print_test "Verifying stored entry uses location as transaction"
    RETRIEVE_RESPONSE=$(curl -s "${LOGGER_ENDPOINT}?source=sensor-03&limit=1")
    if echo "$RETRIEVE_RESPONSE" | grep -q "Building A, Room 101"; then
        print_success "Entry stored with location as transaction"
    else
        print_failure "Entry not stored with location as transaction"
        echo "Retrieve response: $RETRIEVE_RESPONSE"
    fi
else
    print_failure "Failed to create log entry with location"
    echo "Response: $RESPONSE"
fi

# Test 15: Retrieve location entries
print_header "Test 15: Retrieve Location Entries"
print_test "GET /api/logger?source=sensor-03&name=location"

RESPONSE=$(curl -s "${LOGGER_ENDPOINT}?source=sensor-03&name=location")

if check_response "$RESPONSE" "success" "true"; then
    if echo "$RESPONSE" | grep -q "set_location" && echo "$RESPONSE" | grep -q "Building A, Room 101"; then
        print_success "Location entries retrieved successfully"
    else
        print_failure "Location entries not found"
    fi
    echo "Response: $RESPONSE"
else
    print_failure "Failed to retrieve location entries"
    echo "Response: $RESPONSE"
fi

# Test 16: Update location (multiple set_location calls)
print_header "Test 16: Update Location"
print_test "POST /api/logger - Update location for sensor-03"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T11:20:00",
        "transaction": "set_location",
        "name": "location",
        "value": "Building B, Room 205",
        "source": "sensor-03"
    }')

if check_response "$RESPONSE" "success" "true"; then
    print_success "Location updated"
    echo "Response: $RESPONSE"
    
    # Verify new location is used in subsequent entries
    print_test "POST /api/logger - Log entry after location update (should use new location)"
    UPDATE_RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d '{
            "datetime": "2025-01-15T11:25:00",
            "transaction": "logging",
            "name": "humidity",
            "value": 65.0,
            "source": "sensor-03"
        }')
    
    if echo "$UPDATE_RESPONSE" | grep -q "Building B, Room 205"; then
        print_success "New location used in subsequent entry"
    else
        print_failure "New location not used"
        echo "Response: $UPDATE_RESPONSE"
    fi
else
    print_failure "Failed to update location"
    echo "Response: $RESPONSE"
fi

# Test 17: Set location for multiple devices
print_header "Test 17: Multiple Device Locations"
print_test "POST /api/logger - Set location for sensor-04"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T11:30:00",
        "transaction": "set_location",
        "name": "location",
        "value": "Warehouse Zone 3",
        "source": "sensor-04"
    }')

if check_response "$RESPONSE" "success" "true"; then
    print_success "Location set for sensor-04"
    echo "Response: $RESPONSE"
    
    # Create log entry for sensor-04
    print_test "POST /api/logger - Log entry for sensor-04 (should use its location)"
    SENSOR4_RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d '{
            "datetime": "2025-01-15T11:35:00",
            "transaction": "logging",
            "name": "pressure",
            "value": 1013.25,
            "source": "sensor-04"
        }')
    
    if echo "$SENSOR4_RESPONSE" | grep -q "Warehouse Zone 3"; then
        print_success "Sensor-04 uses its own location"
    else
        print_failure "Sensor-04 location not used"
        echo "Response: $SENSOR4_RESPONSE"
    fi
    
    # Verify sensor-03 still uses its location
    print_test "POST /api/logger - Log entry for sensor-03 (should still use Building B location)"
    SENSOR3_RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d '{
            "datetime": "2025-01-15T11:36:00",
            "transaction": "logging",
            "name": "temperature",
            "value": 21.0,
            "source": "sensor-03"
        }')
    
    if echo "$SENSOR3_RESPONSE" | grep -q "Building B, Room 205"; then
        print_success "Sensor-03 still uses its location (locations are device-specific)"
    else
        print_failure "Sensor-03 location incorrect"
        echo "Response: $SENSOR3_RESPONSE"
    fi
else
    print_failure "Failed to set location for sensor-04"
    echo "Response: $RESPONSE"
fi

# Test 18: Device without location uses original transaction
print_header "Test 18: Device Without Location"
print_test "POST /api/logger - Log entry for device without set_location"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T11:40:00",
        "transaction": "logging",
        "name": "voltage",
        "value": 12.5,
        "source": "sensor-05"
    }')

if check_response "$RESPONSE" "success" "true"; then
    # Should not have location field
    if ! echo "$RESPONSE" | grep -q "location"; then
        print_success "Device without location uses original transaction (no location field)"
    else
        print_failure "Location field present for device without set_location"
    fi
    echo "Response: $RESPONSE"
else
    print_failure "Failed to create log entry"
    echo "Response: $RESPONSE"
fi

# Test 19: set_location transaction itself should not be modified
print_header "Test 19: Set Location Transaction Preservation"
print_test "POST /api/logger - set_location transaction should not be modified"

RESPONSE=$(curl -s -X POST "${LOGGER_ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d '{
        "datetime": "2025-01-15T11:45:00",
        "transaction": "set_location",
        "name": "location",
        "value": "Test Location",
        "source": "sensor-06"
    }')

if check_response "$RESPONSE" "success" "true"; then
    # set_location transactions should not have location field in response
    if ! echo "$RESPONSE" | grep -q "\"location\""; then
        print_success "set_location transaction preserved (not modified)"
    else
        print_failure "set_location transaction was modified"
    fi
    echo "Response: $RESPONSE"
    
    # Verify stored entry has transaction="set_location"
    print_test "Verifying stored set_location entry"
    RETRIEVE_RESPONSE=$(curl -s "${LOGGER_ENDPOINT}?source=sensor-06&name=location")
    if echo "$RETRIEVE_RESPONSE" | grep -q "set_location"; then
        print_success "set_location transaction stored correctly"
    else
        print_failure "set_location transaction not stored correctly"
        echo "Retrieve response: $RETRIEVE_RESPONSE"
    fi
else
    print_failure "Failed to create set_location entry"
    echo "Response: $RESPONSE"
fi

# Summary
print_header "Test Summary"
echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
echo -e "${RED}Failed:${NC} $TESTS_FAILED"
echo -e "${BLUE}Total:${NC} $((TESTS_PASSED + TESTS_FAILED))"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed ✗${NC}"
    exit 1
fi
