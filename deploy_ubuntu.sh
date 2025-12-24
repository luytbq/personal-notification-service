#!/bin/bash
#
# Deploy Personal Notification Service to Ubuntu Server
#

set -e

# Configuration
SERVER_HOST="your-user@your-server"
REMOTE_DIR="/opt/pns"
LOG_DIR="/var/log/pns"
SERVICE_NAME="pns"

# Build
echo "Building Linux binary..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/server

# Deploy
echo "Deploying to $SERVER_HOST..."
ssh $SERVER_HOST "sudo systemctl stop $SERVICE_NAME 2>/dev/null || true"
ssh $SERVER_HOST "sudo mkdir -p $REMOTE_DIR $LOG_DIR && sudo chown www-data:www-data $LOG_DIR"
scp server .env $SERVER_HOST:$REMOTE_DIR/
scp pns.service $SERVER_HOST:/tmp/
scp pns.logrotate $SERVER_HOST:/tmp/
ssh $SERVER_HOST "sudo mv /tmp/pns.service /etc/systemd/system/"
ssh $SERVER_HOST "sudo mv /tmp/pns.logrotate /etc/logrotate.d/pns"
ssh $SERVER_HOST "sudo chown -R www-data:www-data $REMOTE_DIR"

# Install and start service
echo "Starting service..."
ssh $SERVER_HOST "sudo systemctl daemon-reload && sudo systemctl enable $SERVICE_NAME && sudo systemctl restart $SERVICE_NAME"

# Cleanup
rm -f server

echo "Done! Check status: ssh $SERVER_HOST 'sudo systemctl status $SERVICE_NAME'"
echo "Logs: ssh $SERVER_HOST 'tail -f $LOG_DIR/pns.log'"
