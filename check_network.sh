#!/bin/bash

# Network diagnostic script for go-logger server
# Run this to check if your server is accessible from remote clients

echo "=== Go Logger Network Diagnostics ==="
echo ""

# Check if server is running
echo "1. Checking if server is running..."
if pgrep -f logger-server > /dev/null; then
    echo "   ✅ Server process found"
    SERVER_PID=$(pgrep -f logger-server | head -1)
    echo "   PID: $SERVER_PID"
else
    echo "   ❌ Server process not found"
    echo "   Start the server with: ./logger-server"
    exit 1
fi

echo ""

# Check what port the server is listening on
echo "2. Checking listening ports..."
if command -v ss > /dev/null; then
    LISTENING=$(ss -tlnp 2>/dev/null | grep 8765 || echo "")
elif command -v netstat > /dev/null; then
    LISTENING=$(netstat -tlnp 2>/dev/null | grep 8765 || echo "")
else
    echo "   ⚠️  Cannot check (ss or netstat not available)"
    LISTENING=""
fi

if [ -n "$LISTENING" ]; then
    if echo "$LISTENING" | grep -q "0.0.0.0:8765"; then
        echo "   ✅ Server is listening on 0.0.0.0:8765 (all interfaces)"
    elif echo "$LISTENING" | grep -q "127.0.0.1:8765"; then
        echo "   ❌ Server is only listening on 127.0.0.1:8765 (localhost only)"
        echo "   Fix: Restart server with HOST=0.0.0.0 ./logger-server"
    else
        echo "   ⚠️  Server listening on: $LISTENING"
    fi
else
    echo "   ⚠️  Could not determine listening address"
fi

echo ""

# Get server IP addresses
echo "3. Server IP addresses (use these in Arduino code):"
if command -v hostname > /dev/null; then
    IPS=$(hostname -I 2>/dev/null | tr ' ' '\n' | grep -v '^$')
    if [ -n "$IPS" ]; then
        for ip in $IPS; do
            echo "   - $ip"
        done
    else
        echo "   ⚠️  Could not determine IP addresses"
    fi
else
    echo "   ⚠️  hostname command not available"
fi

echo ""

# Check firewall
echo "4. Checking firewall status..."
if command -v firewall-cmd > /dev/null; then
    if sudo firewall-cmd --state > /dev/null 2>&1; then
        if sudo firewall-cmd --query-port=8765/tcp > /dev/null 2>&1; then
            echo "   ✅ Port 8765 is allowed in firewalld"
        else
            echo "   ❌ Port 8765 is NOT allowed in firewalld"
            echo "   Fix: sudo firewall-cmd --permanent --add-port=8765/tcp && sudo firewall-cmd --reload"
        fi
    else
        echo "   ℹ️  firewalld is not active"
    fi
elif command -v ufw > /dev/null; then
    UFW_STATUS=$(sudo ufw status 2>/dev/null | head -1)
    if echo "$UFW_STATUS" | grep -q "Status: active"; then
        if sudo ufw status | grep -q "8765"; then
            echo "   ✅ Port 8765 is allowed in ufw"
        else
            echo "   ❌ Port 8765 is NOT allowed in ufw"
            echo "   Fix: sudo ufw allow 8765/tcp"
        fi
    else
        echo "   ℹ️  ufw is not active"
    fi
else
    echo "   ⚠️  Could not check firewall (firewall-cmd or ufw not available)"
fi

echo ""

# Test local connection
echo "5. Testing local connection..."
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8765/health > /dev/null 2>&1; then
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8765/health)
    if [ "$HTTP_CODE" = "200" ]; then
        echo "   ✅ Local connection works (HTTP $HTTP_CODE)"
    else
        echo "   ⚠️  Local connection returned HTTP $HTTP_CODE"
    fi
else
    echo "   ❌ Local connection failed"
fi

echo ""

# Summary
echo "=== Summary ==="
echo ""
echo "For Arduino to connect, ensure:"
echo "  1. Server is listening on 0.0.0.0:8765 (not 127.0.0.1)"
echo "  2. Firewall allows port 8765"
echo "  3. Arduino uses one of the IP addresses shown above (not localhost)"
echo ""
echo "Example Arduino URL:"
if [ -n "$IPS" ]; then
    FIRST_IP=$(echo $IPS | head -1)
    echo "  http://$FIRST_IP:8765/log"
else
    echo "  http://<server-ip>:8765/log"
fi
echo ""
