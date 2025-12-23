#!/bin/bash
#
# Deploy Personal Notification Service to Ubuntu VPS
#

set -e

# Configuration
VPS_HOST="root@luytbq.site"
REMOTE_DIR="/opt/pns"
LOG_DIR="/var/log/pns"
SERVICE_NAME="pns"

# Build
echo "Building Linux binary..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/server

# Deploy
echo "Deploying to $VPS_HOST..."
ssh $VPS_HOST "sudo systemctl stop $SERVICE_NAME 2>/dev/null || true"
ssh $VPS_HOST "sudo mkdir -p $REMOTE_DIR $LOG_DIR && sudo chown www-data:www-data $LOG_DIR"
scp server .env $VPS_HOST:$REMOTE_DIR/
scp pns.service $VPS_HOST:/tmp/
scp pns.logrotate $VPS_HOST:/tmp/
ssh $VPS_HOST "sudo mv /tmp/pns.service /etc/systemd/system/"
ssh $VPS_HOST "sudo mv /tmp/pns.logrotate /etc/logrotate.d/pns"
ssh $VPS_HOST "sudo chown -R www-data:www-data $REMOTE_DIR"

# Install and start service
echo "Starting service..."
ssh $VPS_HOST "sudo systemctl daemon-reload && sudo systemctl enable $SERVICE_NAME && sudo systemctl restart $SERVICE_NAME"

# Cleanup
rm -f server

echo "Done! Check status: ssh $VPS_HOST 'sudo systemctl status $SERVICE_NAME'"
echo "Logs: ssh $VPS_HOST 'tail -f $LOG_DIR/pns.log'"
