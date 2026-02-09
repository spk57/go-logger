# Go Logger API

A lightweight HTTP API server for logging data entries from Arduino and other remote devices. This is a Go port of the Julia `logger.jl` API.

## Features

- CSV-based persistent storage
- Thread-safe concurrent access
- Simple REST API
- CORS enabled for browser/device access
- Arduino-friendly quick logging endpoint

## Building

```bash
cd go-logger
go build -o logger-server .
```

## Running

```bash
# Default port 8765
./logger-server

# Custom port
PORT=3000 ./logger-server

# Custom log file path
LOG_FILE=/path/to/logs.csv ./logger-server

# Enable debug mode (logs all connection requests)
./logger-server -d

# Combine options
PORT=3000 LOG_FILE=/path/to/logs.csv ./logger-server -d
```

### Debug Mode

Use the `-d` flag to enable debug mode, which logs all incoming connection requests to the console. This is helpful for troubleshooting connection issues with remote clients.

**Debug output includes:**
- Timestamp of each request
- HTTP method (GET, POST, etc.)
- URL path
- Client IP address and port
- Query parameters (if any)
- Content-Type header (for POST/PUT requests)

**Example debug output:**
```
[DEBUG] 2025-01-24 15:30:45 - GET /api/logger from 192.168.1.100:52341
[DEBUG]   Query: source=sensor-01&limit=10

[DEBUG] 2025-01-24 15:30:50 - POST /api/logger from 192.168.1.100:52342
[DEBUG]   Content-Type: application/json
```

**When to use debug mode:**
- Troubleshooting why clients can't connect
- Verifying which endpoints clients are hitting
- Checking what parameters are being sent
- Monitoring connection patterns
- Diagnosing network connectivity issues

## API Endpoints

### Add Log Entry

**POST /log** (JSON body)
```bash
curl -X POST http://localhost:8765/log \
  -H "Content-Type: application/json" \
  -d '{"name":"temperature","value":"23.5","source":"arduino-1"}'
```

**GET /quick** (Query parameters - Arduino friendly)
```bash
curl "http://localhost:8765/quick?name=temperature&value=23.5&source=arduino-1"
```

Response:
```json
{
  "success": true,
  "message": "Log entry created successfully",
  "id": 1
}
```

### Get Log Entries

**GET /log** or **GET /logs**

Query parameters:
- `limit` - Max entries to return (default: 100)
- `offset` - Skip N entries (default: 0)
- `source` - Filter by source
- `name` - Filter by name

```bash
# Get all entries
curl http://localhost:8765/log

# Filter by source
curl "http://localhost:8765/log?source=arduino-1&limit=50"
```

Response:
```json
{
  "success": true,
  "entries": [
    {
      "id": 1,
      "datetime": "2026-01-12T10:30:00Z",
      "name": "temperature",
      "value": "23.5",
      "source": "arduino-1",
      "created_at": "2026-01-12T10:30:00Z"
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0
}
```

### Get Statistics

**GET /stats**

```bash
curl http://localhost:8765/stats
```

Response:
```json
{
  "success": true,
  "total_entries": 150,
  "unique_sources": 3,
  "unique_names": 5,
  "sources": ["arduino-1", "arduino-2", "esp32"],
  "names": ["temperature", "humidity", "pressure", "voltage", "light"]
}
```

### Clear All Entries

**DELETE /log**

```bash
curl -X DELETE http://localhost:8765/log
```

### Health Check

**GET /health**

```bash
curl http://localhost:8765/health
```

## Arduino Example (ESP8266/ESP32)

```cpp
#include <WiFi.h>
#include <HTTPClient.h>

const char* serverUrl = "http://192.168.1.100:8765/quick";

void logValue(const char* name, float value, const char* source) {
  if (WiFi.status() == WL_CONNECTED) {
    HTTPClient http;
    
    String url = String(serverUrl) + 
                 "?name=" + name + 
                 "&value=" + String(value) + 
                 "&source=" + source;
    
    http.begin(url);
    int httpCode = http.GET();
    
    if (httpCode > 0) {
      Serial.println("Log sent successfully");
    }
    http.end();
  }
}

void loop() {
  float temp = readTemperature();
  logValue("temperature", temp, "arduino-living-room");
  delay(60000); // Log every minute
}
```

## CSV Format

Data is stored in CSV format with the following columns:

| Column | Description |
|--------|-------------|
| id | Auto-incrementing entry ID |
| datetime | Timestamp of the measurement |
| name | Measurement name (e.g., "temperature") |
| value | Measured value |
| source | Device identifier |
| created_at | When the entry was created on the server |

## Troubleshooting Remote Connections

If local `curl` commands work but remote Arduino clients cannot connect:

### Enable Debug Mode

Start the server with debug mode to see all connection attempts:
```bash
./logger-server -d
```

This will show you:
- Which clients are trying to connect
- What endpoints they're accessing
- What parameters they're sending
- When connection attempts occur

If you don't see any debug output when a client tries to connect, the request isn't reaching the server (likely a network/firewall issue).

### Quick Diagnostic

Run the network diagnostic script:
```bash
./check_network.sh
```

This will check:
- Server process status
- Network binding (should be `0.0.0.0:8765`, not `127.0.0.1:8765`)
- Server IP addresses
- Firewall configuration
- Local connectivity

### Common Issues

1. **Firewall blocking port 8765**
   - Linux (firewalld): `sudo firewall-cmd --permanent --add-port=8765/tcp && sudo firewall-cmd --reload`
   - Linux (ufw): `sudo ufw allow 8765/tcp`

2. **Arduino using wrong URL**
   - ❌ `http://localhost:8765/log` (won't work remotely)
   - ✅ `http://192.168.1.100:8765/log` (use actual server IP)

3. **Server not binding to all interfaces**
   - The server binds to `0.0.0.0` by default
   - Explicitly set: `HOST=0.0.0.0 ./logger-server`

4. **No connection attempts visible in debug mode**
   - If debug mode shows no output when clients try to connect, the requests aren't reaching the server
   - Check firewall rules, network routing, and that the client is using the correct IP address
   - Verify the server is listening on the expected interface: `netstat -tuln | grep 8765`

5. **Connection attempts visible but failing**
   - If debug mode shows connection attempts but they fail, check:
     - Server logs for error messages
     - Request format (method, headers, body)
     - CORS issues (though CORS is enabled by default)

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed troubleshooting steps.
