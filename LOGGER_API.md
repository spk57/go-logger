# Logger API Documentation

## Overview

The Logger API provides endpoints for storing and retrieving time-series log data with metadata. Each log entry contains a timestamp, transaction identifier, name, value, source identifier, and location. The location field is automatically populated from `set_location` transactions when available.

## Transaction Types

The API supports different transaction types to categorize log entries:

- **`logging`**: Standard data logging (default for sensor readings, metrics, etc.)
- **`set_location`**: Set or update the location for a logging device (see [Device Location Management](#device-location-management))
- **`note`**: Log notes about changes in the environment for measurement devices (see [Note Transactions](#note-transactions))

Transaction types are stored as part of each log entry and can be used to categorize and filter data. The transaction field is optional but recommended for better data organization.

## Location Field

Each log entry includes a `location` field that indicates where the measurement device is located. The location is automatically populated from the most recent `set_location` transaction for that device:

- **Automatic Population**: When a device logs an entry, the system checks for the most recent `set_location` transaction for that device's source identifier
- **Location Updates**: When you create a new `set_location` transaction, all subsequent entries from that device will use the new location
- **Empty Location**: If no `set_location` transaction exists for a device, the location field will be an empty string
- **Storage**: The location is stored in the CSV file and included in all API responses

**Example Flow:**
1. Set location: `POST /api/logger` with `transaction: "set_location"`, `name: "location"`, `value: "Building A, Room 101"`, `source: "sensor-01"`
2. Log data: `POST /api/logger` with `source: "sensor-01"` and any other transaction
3. Retrieve: `GET /api/logger?source=sensor-01` - all entries will include `"location": "Building A, Room 101"`

## Endpoints

### POST /api/logger

Create a new log entry.

**Request Body:**
```json
{
  "datetime": "2025-01-01T10:30:00",
  "transaction": "logging",
  "name": "temperature",
  "value": 23.5,
  "source": "sensor-01"
}
```

**Request Fields:**
- `datetime` (required): ISO 8601 formatted datetime string
- `transaction` (optional): Transaction identifier (e.g., "logging", "set_location", "note"). Defaults to empty string if not provided.
- `name` (required for non-note transactions): String identifier for the log entry
- `value` (required for non-note transactions): Any value (number, string, boolean, etc.)
- `source` (required): String identifying the source/origin
- `note` (required for "note" transactions): Note content/text (see [Note Transactions](#note-transactions))

**Note:** The `location` field is automatically populated from the most recent `set_location` transaction for the device. You do not need to include it in the request. If no location has been set for this device, the location will be an empty string.

**Special Transaction Types:**
- For `note` transactions: Use the `note` field instead of `name` and `value`
- For `set_location` transactions: Use `name: "location"` and `value: "<location string>"`

**Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Log entry created successfully",
  "id": 1
}
```

**Error Response (400 Bad Request):**
```json
{
  "success": false,
  "message": "Missing required field: datetime"
}
```

**Example:**
```bash
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-01T10:30:00",
    "transaction": "logging",
    "name": "temperature",
    "value": 23.5,
    "source": "sensor-01"
  }'
```

---

### GET /api/logger

Retrieve log entries with optional filtering and pagination.

**Query Parameters:**
- `limit` (optional): Maximum number of entries to return (default: 100)
- `offset` (optional): Number of entries to skip (default: 0)
- `source` (optional): Filter by source
- `name` (optional): Filter by name

**Success Response (200 OK):**
```json
{
  "success": true,
  "entries": [
    {
      "id": 1,
      "transaction": "logging",
      "datetime": "2025-01-01T10:30:00",
      "name": "temperature",
      "value": 23.5,
      "source": "sensor-01",
      "location": "Building A, Room 101",
      "created_at": "2025-01-01T10:30:05"
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0
}
```

**Note:** The `location` field is automatically populated from the most recent `set_location` transaction for the device. If no location has been set, the field will be an empty string.

**Examples:**

Get all entries:
```bash
curl http://localhost:8765/api/logger
```

Get entries with pagination:
```bash
curl "http://localhost:8765/api/logger?limit=10&offset=20"
```

Filter by source:
```bash
curl "http://localhost:8765/api/logger?source=sensor-01"
```

Filter by name:
```bash
curl "http://localhost:8765/api/logger?name=temperature"
```

Combine filters:
```bash
curl "http://localhost:8765/api/logger?source=sensor-01&name=temperature&limit=50"
```

---

### GET /api/logger/stats

Get statistics about logged entries.

**Success Response (200 OK):**
```json
{
  "success": true,
  "total_entries": 150,
  "unique_sources": 5,
  "unique_names": 10,
  "sources": ["sensor-01", "sensor-02", "api", "manual", "system"],
  "names": ["temperature", "humidity", "pressure", "status", ...]
}
```

**Example:**
```bash
curl http://localhost:8765/api/logger/stats
```

---

## Use Cases

### IoT Sensor Data Logging

```bash
# Log temperature reading
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-01T10:30:00",
    "name": "temperature",
    "value": 23.5,
    "source": "sensor-01"
  }'

# Log humidity reading
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-01T10:30:00",
    "name": "humidity",
    "value": 65.2,
    "source": "sensor-01"
  }'

# Retrieve all sensor-01 readings
curl "http://localhost:8765/api/logger?source=sensor-01"
```

### Application Event Logging

```bash
# Log application event
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-01T10:30:00",
    "name": "user_login",
    "value": "success",
    "source": "web-app"
  }'

# Log error event
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-01T10:35:00",
    "name": "api_error",
    "value": "timeout",
    "source": "backend-service"
  }'
```

### System Metrics

```bash
# Log CPU usage
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-01T10:30:00",
    "name": "cpu_usage",
    "value": 45.2,
    "source": "server-01"
  }'

# Get all metrics for server-01
curl "http://localhost:8765/api/logger?source=server-01"
```

### Device Location Management

The `set_location` transaction is used to set or update the location for a specific logging device. Once a location is set, all subsequent log entries from that device will automatically include the location in the `location` field.

**How Location Works:**
1. Use `set_location` transaction to set a location for a device
2. All future log entries from that device will automatically have the `location` field populated
3. The location is stored in the CSV file and included in API responses
4. You can update the location by creating a new `set_location` transaction (most recent one is used)

**Transaction Fields:**
- `transaction`: Must be `"set_location"`
- `name`: Should be `"location"` (identifies the field being set)
- `value`: The location string (e.g., "Building A, Room 101", "Warehouse Zone 3", "GPS: 40.7128,-74.0060")
- `source`: The device identifier (required)

**Example:**
```bash
# Set location for a logging device
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-01T10:30:00",
    "transaction": "set_location",
    "name": "location",
    "value": "Building A, Room 101",
    "source": "sensor-01"
  }'

# Now all subsequent entries from sensor-01 will include this location
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-01T10:35:00",
    "transaction": "logging",
    "name": "temperature",
    "value": 23.5,
    "source": "sensor-01"
  }'

# The response will include the location field:
# {
#   "success": true,
#   "message": "Log entry created successfully",
#   "id": 2
# }
# And when retrieving, the entry will have:
# {
#   "id": 2,
#   "transaction": "logging",
#   "name": "temperature",
#   "value": "23.5",
#   "source": "sensor-01",
#   "location": "Building A, Room 101",
#   ...
# }

# Update location for a device
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-15T14:20:00",
    "transaction": "set_location",
    "name": "location",
    "value": "Building B, Room 205",
    "source": "sensor-01"
  }'

# All new entries from sensor-01 will now use "Building B, Room 205"
```

**Response:**
```json
{
  "success": true,
  "message": "Log entry created successfully",
  "id": 42
}
```

**Retrieving Location Entries:**
```bash
# Get all location entries for a specific device
curl "http://localhost:8765/api/logger?source=sensor-01&name=location"

# Get the most recent location for a device
curl "http://localhost:8765/api/logger?source=sensor-01&name=location&limit=1"

# Get all entries for a device (will include location field)
curl "http://localhost:8765/api/logger?source=sensor-01"
```

### Note Transactions

The `note` transaction allows you to log notes about changes in the environment for measurement devices. Notes are useful for documenting events, changes, or observations related to device operation.

**Transaction Fields:**
- `transaction`: Must be `"note"`
- `note` (required): The note content/text
- `source`: The device identifier (required)
- `datetime`: Timestamp of when the note was created

**Note:** For `note` transactions, the `note` field is required instead of `name` and `value`. The system automatically sets `name` to `"note"` and stores the note content in the `value` field.

**Example:**
```bash
# Log a note about a device
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-15T14:30:00",
    "transaction": "note",
    "note": "Device moved to new location due to maintenance",
    "source": "sensor-01"
  }'

# Log another note
curl -X POST http://localhost:8765/api/logger \
  -H "Content-Type: application/json" \
  -d '{
    "datetime": "2025-01-15T15:00:00",
    "transaction": "note",
    "note": "Calibration performed - all readings verified",
    "source": "sensor-01"
  }'

# Retrieve all notes for a device
curl "http://localhost:8765/api/logger?source=sensor-01&transaction=note"

# Or filter by name
curl "http://localhost:8765/api/logger?source=sensor-01&name=note"
```

**Response:**
```json
{
  "success": true,
  "message": "Log entry created successfully",
  "id": 45
}
```

**Retrieved Note Entry:**
```json
{
  "id": 45,
  "transaction": "note",
  "datetime": "2025-01-15T14:30:00",
  "name": "note",
  "value": "Device moved to new location due to maintenance",
  "source": "sensor-01",
  "location": "Building A, Room 101",
  "created_at": "2025-01-15T14:30:05"
}
```

**Using Query Parameters:**
```bash
# Quick note via GET request
curl "http://localhost:8765/quick?transaction=note&note=Device%20restarted&source=sensor-01"
```

---

## Python Client Example

```python
import requests
from datetime import datetime

class LoggerClient:
    def __init__(self, base_url="http://localhost:8765"):
        self.base_url = base_url
    
    def log(self, name, value, source, dt=None):
        """Create a log entry"""
        if dt is None:
            dt = datetime.now()
        
        payload = {
            "datetime": dt.isoformat(),
            "name": name,
            "value": value,
            "source": source
        }
        
        response = requests.post(
            f"{self.base_url}/api/logger",
            json=payload
        )
        return response.json()
    
    def get_entries(self, limit=100, offset=0, source=None, name=None):
        """Retrieve log entries"""
        params = {"limit": limit, "offset": offset}
        if source:
            params["source"] = source
        if name:
            params["name"] = name
        
        response = requests.get(
            f"{self.base_url}/api/logger",
            params=params
        )
        return response.json()
    
    def get_stats(self):
        """Get logger statistics"""
        response = requests.get(f"{self.base_url}/api/logger/stats")
        return response.json()

# Usage
client = LoggerClient()

# Log some data
client.log("temperature", 23.5, "sensor-01")
client.log("humidity", 65.2, "sensor-01")

# Retrieve entries
entries = client.get_entries(source="sensor-01")
print(f"Found {entries['total']} entries")

# Access location field from entries
for entry in entries['entries']:
    print(f"Entry: {entry['name']} = {entry['value']} at {entry.get('location', 'unknown location')}")

# Get stats
stats = client.get_stats()
print(f"Total entries: {stats['total_entries']}")
```

---

## Julia Client Example

```julia
using HTTP, JSON, Dates

struct LoggerClient
    base_url::String
end

function log_entry(client::LoggerClient, transaction::String, name::String, value, source::String; dt::DateTime=now())
    payload = Dict(
        "datetime" => Dates.format(dt, "yyyy-mm-ddTHH:MM:SS"),
        "transaction" => transaction,
        "name" => name,
        "value" => value,
        "source" => source
    )
    
    response = HTTP.post(
        "$(client.base_url)/api/logger",
        body=JSON.json(payload),
        headers=Dict("Content-Type" => "application/json")
    )
    
    return JSON.parse(String(response.body))
end

function get_entries(client::LoggerClient; limit=100, offset=0, source=nothing, name=nothing)
    params = ["limit=$limit", "offset=$offset"]
    !isnothing(source) && push!(params, "source=$source")
    !isnothing(name) && push!(params, "name=$name")
    
    url = "$(client.base_url)/api/logger?" * join(params, "&")
    response = HTTP.get(url)
    
    return JSON.parse(String(response.body))
end

# Usage
client = LoggerClient("http://localhost:8765")

# Log data
log_entry(client, "logging", "temperature", 23.5, "sensor-01")

# Retrieve entries
entries = get_entries(client, source="sensor-01")
println("Found $(entries["total"]) entries")

# Access location field from entries
for entry in entries["entries"]
    location = get(entry, "location", "unknown location")
    println("Entry: $(entry["name"]) = $(entry["value"]) at $location")
end
```

---

## Data Storage

The logger stores data persistently in a CSV file (`logger.csv`) located in the project root directory.

**Benefits:**
- ✅ **Persistent**: Data survives server restarts
- ✅ **Simple**: No database setup required
- ✅ **Portable**: Easy to backup, transfer, or analyze
- ✅ **Human-readable**: Can be opened in Excel, pandas, etc.
- ✅ **Thread-safe**: Concurrent access protected with locks

**CSV File Structure:**
```csv
id,transaction,datetime,name,value,source,location,created_at
1,logging,2025-01-01T10:30:00,temperature,23.5,sensor-01,"Building A, Room 101",2025-01-01T10:30:05
2,logging,2025-01-01T10:31:00,humidity,65.2,sensor-01,"Building A, Room 101",2025-01-01T10:31:03
3,set_location,2025-01-01T10:00:00,location,"Building A, Room 101",sensor-01,"Building A, Room 101",2025-01-01T10:00:02
4,note,2025-01-01T11:00:00,note,"Device moved to new location",sensor-01,"Building A, Room 101",2025-01-01T11:00:05
```

**CSV Columns:**
- `id`: Unique identifier for the log entry
- `transaction`: Transaction type (e.g., "logging", "set_location")
- `datetime`: Timestamp of the logged event
- `name`: Name identifier for the log entry
- `value`: The logged value
- `source`: Source/device identifier
- `location`: Location of the measurement device (populated from `set_location` transactions)
- `created_at`: Timestamp when the entry was created in the system

**Considerations:**
- Suitable for small to medium datasets (thousands to hundreds of thousands of entries)
- For very large datasets or high-frequency logging, consider a database
- CSV file grows over time - implement data retention/archival as needed
- Backup the `logger.csv` file regularly

**For production with high volume**, consider migrating to:
- PostgreSQL (relational database)
- InfluxDB or TimescaleDB (time-series optimized)
- SQLite (embedded database)

---

## Best Practices

1. **Use ISO 8601 datetime format**: Always use `YYYY-MM-DDTHH:MM:SS` format
2. **Consistent naming**: Use consistent names for similar log types
3. **Source identification**: Use clear, unique source identifiers
4. **Set device locations**: Use `set_location` transactions to track where devices are located
5. **Location updates**: Update device locations when devices are moved using new `set_location` transactions
6. **Pagination**: Use limit/offset for large datasets
7. **Filtering**: Filter by source or name to reduce data transfer
8. **Regular cleanup**: Implement data retention policies for production

---

## Error Handling

All endpoints return consistent error responses:

```json
{
  "success": false,
  "message": "Description of the error"
}
```

Common errors:
- Missing required fields → 400 Bad Request
- Invalid datetime format → 400 Bad Request
- Server errors → 500 Internal Server Error

---

## Performance Considerations

- In-memory storage is fast but limited by RAM
- Consider implementing pagination for large datasets
- Filter early to reduce data transfer
- Monitor memory usage with many entries
- Implement data archival/cleanup for long-running systems

