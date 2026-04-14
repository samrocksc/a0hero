#!/bin/bash
cd /Users/sam/GitHub/work/a0hero

# Build
go build -o a0hero ./cmd/a0hero

# Setup config
rm -f ~/.config/a0hero/acul-tryout.yaml
cat > ~/.config/a0home/a0hero/acul-tryout.yaml << 'EOF' 2>/dev/null || mkdir -p ~/.config/a0hero
EOF
cat > ~/.config/a0hero/acul-tryout.yaml << 'EOF'
name: acul-tryout
domain: acul-tryout.cic-demo-platform.auth0app.com
client_id: tdxxRErnhXjAJdW90nzDVceA5oFH7HZx
client_secret: dCYiiSZKWXZ7R7zXSDRATYNoo6KtrB0-Ynf2SDpl6G72uourXXyol-uo355Td8it
EOF

# Clear logs
rm -f logs/*.log

# Run with script to fake TTY
echo "Running a0hero with PTY..."
script -q -c "echo 'l'; sleep 0.5; echo 'l'; sleep 0.5; echo 'e'; sleep 3; echo 'q'" /dev/null ./a0hero --debug 2>&1 | tee test_output.log &
PID=$!
sleep 5
kill $PID 2>/dev/null || true

echo "=== LOG OUTPUT ==="
cat logs/*.log 2>/dev/null || echo "No logs"
