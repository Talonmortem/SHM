#!/bin/bash
set -euo pipefail

# Configuration
REMOTE_USER="talonmortem"
REMOTE_HOST="146.19.128.211"
REMOTE_DIR="/app/shm"
PROJECT_DIR="."
PUBLIC_PORT="8082"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Check if SSH key is set up or prompt for password
echo "Checking SSH access to $REMOTE_USER@$REMOTE_HOST..."
if ! ssh -o BatchMode=yes "$REMOTE_USER@$REMOTE_HOST" true 2>/dev/null; then
  echo "SSH key-based authentication not set up. You may be prompted for a password."
fi

# Step 0: Stop docker containers on the remote server
echo "Stopping existing Docker containers on $REMOTE_HOST..."
ssh "$REMOTE_USER@$REMOTE_HOST" "docker compose -f $REMOTE_DIR/docker-compose.yaml down" || true

# Step 1: Copy the entire project directory to the remote server

echo "Copying project directory to $REMOTE_HOST:$REMOTE_DIR..."
rsync -avz --progress --delete --exclude '.git' "$PROJECT_DIR/" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR"
echo -e "${GREEN}Successfully copied project directory to $REMOTE_DIR${NC}"

# Step 2: Start docker compose on the remote server
echo "Starting Docker Compose on $REMOTE_HOST..."
ssh "$REMOTE_USER@$REMOTE_HOST" "cd $REMOTE_DIR && docker compose down && docker compose up -d --build"

# Step 3: Wait for the app to be ready via public endpoint
echo "Waiting for the application to be ready..."
while ! ssh "$REMOTE_USER@$REMOTE_HOST" "curl -fsS http://127.0.0.1:$PUBLIC_PORT/health" | grep -q "ok"; do
  echo -n "."
  sleep 5
done
echo -e "\n${GREEN}Application is ready!${NC}"

echo -e "${GREEN}Deployment completed successfully!${NC}"
