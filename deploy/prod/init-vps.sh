#!/usr/bin/env bash

# ==============================================================================
# VPS Initialization & Security Setup Script
# Target OS: Ubuntu Server 22.04 LTS / 24.04 LTS
# Project: secret-letter (Milestone 5 Production Deployment)
# ==============================================================================

# Exit immediately if a command exits with a non-zero status
set -e

# Define colors for output formatting
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}======================================================================${NC}"
echo -e "${GREEN}    STARTING SECRET-LETTER VPS INITIALIZATION & SECURITY SETUP        ${NC}"
echo -e "${YELLOW}======================================================================${NC}"

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Error: Please run this script as root (sudo ./init-vps.sh).${NC}"
  exit 1
fi

# 1. Update and Upgrade System packages
echo -e "\n${YELLOW}[1/5] Updating system packages...${NC}"
apt-get update -y
apt-get upgrade -y

# 2. Install basic utility dependencies
echo -e "\n${YELLOW}[2/5] Installing basic utilities (git, curl, ufw, etc.)...${NC}"
apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    git \
    ufw \
    htop \
    fail2ban

# 3. Secure VPS with Firewall (UFW)
echo -e "\n${YELLOW}[3/5] Setting up security firewall (UFW)...${NC}"
# Default policies
ufw default deny incoming
ufw default allow outgoing

# Allow standard web and management ports
ufw allow 22/tcp comment 'SSH'
ufw allow 80/tcp comment 'HTTP'
ufw allow 443/tcp comment 'HTTPS'

# Explicitly block Redis standard port from external world (even though Docker handles it, this is best-practice safety)
ufw deny 6379/tcp comment 'Block Redis External'

# Enable firewall without interactive prompt
echo "y" | ufw enable
ufw status verbose

# 4. Install Docker & Docker Compose from official Docker repositories
echo -e "\n${YELLOW}[4/5] Installing Docker Engine & Docker Compose...${NC}"
# Add Docker's official GPG key
mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg --yes

# Set up the stable repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker packages
apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Start and enable Docker service
systemctl enable docker
systemctl start docker

# Verify installations
echo -e "${GREEN}✔ Docker version: $(docker --version)${NC}"
echo -e "${GREEN}✔ Docker Compose version: $(docker compose version)${NC}"

# 5. Provide deployment guide for user
echo -e "\n${YELLOW}[5/5] Setup completed successfully!${NC}"
echo -e "${YELLOW}======================================================================${NC}"
echo -e "${GREEN}                     READY FOR DEPLOYMENT                             ${NC}"
echo -e "${YELLOW}======================================================================${NC}"
echo -e "Next steps to deploy the application:"
echo -e "1. Create your production environment file:"
echo -e "   ${GREEN}cp .env.example .env${NC}"
echo -e "2. Edit ${GREEN}.env${NC} and fill in ${GREEN}APP_ENV=production${NC}, a stable ${GREEN}SECRET_ENCRYPTION_KEY${NC}, and a strong ${GREEN}REDIS_PASSWORD${NC}."
echo -e "3. Build and launch the application using Docker Compose:"
echo -e "   ${GREEN}docker compose up -d --build${NC}"
echo -e "4. Monitor the boot status of Caddy, API, and Redis:"
echo -e "   ${GREEN}docker compose logs -f${NC}"
echo -e "5. Verify the API health and readiness endpoints at:"
echo -e "   ${GREEN}https://api.secret.quorix.io.vn/healthz${NC}"
echo -e "   ${GREEN}https://api.secret.quorix.io.vn/readyz${NC}"
echo -e "${YELLOW}======================================================================${NC}"
