# Troubleshooting Remote Client Connections

If local `curl` commands work but remote Arduino clients cannot connect, check the following:

## 1. Verify Server is Listening on All Interfaces

The server should bind to `0.0.0.0` (all interfaces) by default. Check what the server is actually listening on:

```bash
# Check if server is listening on port 8765
sudo netstat -tlnp | grep 8765
# or
sudo ss -tlnp | grep 8765
```

You should see something like:
```
tcp  0  0  0.0.0.0:8765  0.0.0.0:*  LISTEN  <pid>/logger-server
```

If you see `127.0.0.1:8765` instead of `0.0.0.0:8765`, the server is only listening on localhost.

**Fix:** The server code should use `0.0.0.0` by default. You can also explicitly set:
```bash
HOST=0.0.0.0 ./logger-server
```

## 2. Check Firewall Rules

The most common issue is a firewall blocking incoming connections on port 8765.

### Linux (firewalld)
```bash
# Check if firewall is active
sudo firewall-cmd --state

# Allow port 8765
sudo firewall-cmd --permanent --add-port=8765/tcp
sudo firewall-cmd --reload

# Verify
sudo firewall-cmd --list-ports
```

### Linux (ufw)
```bash
# Check status
sudo ufw status

# Allow port 8765
sudo ufw allow 8765/tcp

# Verify
sudo ufw status numbered
```

### Linux (iptables)
```bash
# Allow port 8765
sudo iptables -A INPUT -p tcp --dport 8765 -j ACCEPT
sudo iptables-save
```

## 3. Verify Server IP Address

Find your server's IP address that the Arduino should connect to:

```bash
# Get IP addresses
ip addr show
# or
hostname -I
# or
ifconfig
```

Look for your network interface (usually `eth0`, `wlan0`, or `enp*`). The Arduino needs to use this IP address, not `localhost` or `127.0.0.1`.

**Example Arduino code:**
```cpp
const char* serverUrl = "http://192.168.1.100:8765/log";  // Use actual server IP
```

## 4. Test Remote Connection from Another Machine

Before troubleshooting the Arduino, test from another computer on the same network:

```bash
# From another machine on the network
curl -X POST http://<server-ip>:8765/log \
  -H "Content-Type: application/json" \
  -d '{"name":"test","value":"1.0","source":"test-client"}'
```

If this works, the issue is with the Arduino configuration. If it doesn't, the issue is with the server/firewall.

## 5. Check Network Connectivity

Ensure the Arduino and server are on the same network:

```bash
# From server, ping Arduino (if you know its IP)
ping <arduino-ip>

# Check if Arduino can reach server
# (You may need to check Arduino serial output or logs)
```

## 6. Verify Port is Not Blocked by Router

Some routers block incoming connections. If Arduino is on a different network:
- Use port forwarding on the router
- Or ensure both devices are on the same local network

## 7. Check Server Logs

The server should show connection attempts. Look for:
- Connection errors
- Request logs
- Any error messages

## 8. Test with Health Endpoint

Test basic connectivity first:

```bash
# From Arduino's network, test health endpoint
curl http://<server-ip>:8765/health
```

This simpler endpoint helps isolate HTTP vs. endpoint-specific issues.

## 9. Common Arduino Issues

### Wrong URL
- ❌ `http://localhost:8765/log` (won't work from remote)
- ❌ `http://127.0.0.1:8765/log` (won't work from remote)
- ✅ `http://192.168.1.100:8765/log` (use actual server IP)

### WiFi Connection
- Ensure Arduino is connected to WiFi
- Check WiFi signal strength
- Verify Arduino can reach other network resources

### HTTP Library Issues
- Some Arduino HTTP libraries have timeout issues
- Try increasing timeout values
- Check for library-specific connection requirements

## 10. Quick Diagnostic Checklist

- [ ] Server is running (`ps aux | grep logger-server`)
- [ ] Server is listening on `0.0.0.0:8765` (not `127.0.0.1:8765`)
- [ ] Firewall allows port 8765
- [ ] Arduino uses correct server IP (not localhost)
- [ ] Arduino and server are on same network
- [ ] Can connect from another machine on network
- [ ] Server logs show connection attempts
- [ ] Arduino WiFi is connected
- [ ] Arduino code has correct URL

## Getting Help

If issues persist, collect:
1. Server startup logs
2. `netstat` or `ss` output showing listening ports
3. Firewall status
4. Server IP address
5. Arduino IP address
6. Network topology (same subnet?)
7. Any error messages from Arduino serial output
