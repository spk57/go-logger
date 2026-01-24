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
```

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

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed troubleshooting steps.
