#!/bin/bash
set -e

cd /Users/sam/GitHub/work/a0hero

echo "=== Building ==="
go build -o a0hero ./cmd/a0hero

echo "=== Setting up config ==="
rm -f ~/.config/a0hero/acul-tryout.yaml
cat > ~/.config/a0hero/acul-tryout.yaml << 'EOF'
name: acul-tryout
domain: acul-tryout.cic-demo-platform.auth0app.com
client_id: tdxxRErnhXjAJdW90nzDVceA5oFH7HZx
client_secret: dCYiiSZKWXZ7R7zXSDRATYNoo6KtrB0-Ynf2SDpl6G72uourXXyol-uo355Td8it
EOF

echo "=== Clearing logs ==="
rm -f logs/*.log

echo "=== Running with script ==="
# Use expect-style script to interact
./a0hero --debug 2>&1 &
APP_PID=$!

echo "App PID: $APP_PID"
sleep 3

# Check if process is running
if kill -0 $APP_PID 2>/dev/null; then
    echo "=== App is running, checking logs ==="
    sleep 2
    cat logs/*.log 2>/dev/null || echo "No log file yet"
else
    echo "=== App exited early ==="
    cat logs/*.log 2>/dev/null
fi

# Kill if still running
kill $APP_PID 2>/dev/null || true
