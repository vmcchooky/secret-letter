# Deployment Guide

This guide covers the complete deployment strategy for the one-time-link application, optimized for portfolio use with low cost and high learning value.

## 1. Deployment Overview

**Architecture Summary:**
- **Frontend**: React app deployed on Vercel
- **Backend**: Single Go binary on Vietnamese VPS (primary)
- **Database**: Self-hosted Redis on same VPS
- **Standby**: Oracle Cloud VPS for failover and learning
- **Domain**: Subdomains of existing `quorix.io.vn`

**Cost Structure:**
- Frontend: $0 (Vercel Hobby plan)
- Domain: $0 (existing `quorix.io.vn`)
- Primary VPS: ~$5-10/month (Vietnamese provider)
- Standby VPS: $0 (Oracle Cloud Free Tier)
- **Total**: ~$5-10/month

## 2. Domain Configuration

### 2.1 DNS Records (PA Vietnam)

Configure these DNS records for `quorix.io.vn`:

```dns
# Keep existing personal website
quorix.io.vn.           A       <existing-vercel-ip>

# Frontend subdomain (CNAME to Vercel)
secret.quorix.io.vn.    CNAME   cname.vercel-dns.com.

# API subdomain (A record to VPS)
api.secret.quorix.io.vn. A      <vietnamese-vps-ip>
```

### 2.2 TTL Settings

Set low TTL for API subdomain to enable fast failover:
- `api.secret.quorix.io.vn`: TTL 300 seconds (5 minutes)
- Other records: TTL 3600 seconds (1 hour)

## 3. Frontend Deployment (Vercel)

### 3.1 Project Setup

1. Create new Vercel project for frontend
2. Set build settings:
   - **Framework**: React
   - **Root Directory**: `frontend/web-app`
   - **Build Command**: `npm run build`
   - **Output Directory**: `dist`

### 3.2 Environment Variables

Configure in Vercel dashboard:

```env
# Production
VITE_API_BASE_URL=https://api.secret.quorix.io.vn

# Development (for preview deployments)
VITE_API_BASE_URL=https://api.secret.quorix.io.vn
```

### 3.3 Custom Domain

1. Add custom domain: `secret.quorix.io.vn`
2. Verify domain ownership
3. Vercel automatically provisions SSL certificate

## 4. Primary VPS Setup (Vietnamese Provider)

### 4.1 VPS Specifications

**Minimum Requirements:**
- **CPU**: 1 vCPU (shared acceptable)
- **RAM**: 1GB
- **Storage**: 20GB SSD
- **Bandwidth**: 1TB/month
- **OS**: Ubuntu 22.04 LTS

### 4.2 Initial Server Hardening

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Create non-root user
sudo adduser deploy
sudo usermod -aG sudo deploy

# Configure SSH key authentication
mkdir -p /home/deploy/.ssh
echo "your-public-key" >> /home/deploy/.ssh/authorized_keys
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh

# Disable password authentication
sudo sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo systemctl restart ssh

# Configure firewall
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow http
sudo ufw allow https
sudo ufw enable
```

### 4.3 Install Dependencies

```bash
# Install Go
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Redis
sudo apt install redis-server -y

# Configure Redis
sudo sed -i 's/bind 127.0.0.1 ::1/bind 127.0.0.1/' /etc/redis/redis.conf
sudo sed -i 's/# requirepass foobared/requirepass your-redis-password/' /etc/redis/redis.conf
sudo systemctl restart redis-server
sudo systemctl enable redis-server

# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy -y
```

### 4.4 Application Deployment

```bash
# Create application directory
sudo mkdir -p /opt/one-time-link
sudo chown deploy:deploy /opt/one-time-link

# Clone repository (or upload binary)
cd /opt/one-time-link
git clone https://github.com/your-username/one-time-link.git .

# Build application
cd backend
go build -o /opt/one-time-link/api ./cmd/api

# Create environment file
sudo tee /opt/one-time-link/.env << EOF
APP_SERVICE_NAME=one-time-link-api
APP_HOST=0.0.0.0
APP_PORT=8080
ALLOWED_ORIGIN=https://secret.quorix.io.vn
EOF

# Create systemd service
sudo tee /etc/systemd/system/one-time-link.service << EOF
[Unit]
Description=One Time Link API
After=network.target redis-server.service
Requires=redis-server.service

[Service]
Type=simple
User=deploy
Group=deploy
WorkingDirectory=/opt/one-time-link
ExecStart=/opt/one-time-link/api
EnvironmentFile=/opt/one-time-link/.env
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable one-time-link
sudo systemctl start one-time-link
```

### 4.5 Caddy Configuration

```bash
# Create Caddyfile
sudo tee /etc/caddy/Caddyfile << EOF
api.secret.quorix.io.vn {
    reverse_proxy localhost:8080
    
    # Security headers
    header {
        X-Content-Type-Options nosniff
        X-Frame-Options DENY
        X-XSS-Protection "1; mode=block"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
    
    # Logging
    log {
        output file /var/log/caddy/api.log
        format json
    }
}
EOF

# Reload Caddy
sudo systemctl reload caddy
```

## 5. Standby VPS Setup (Oracle Cloud)

### 5.1 Oracle Cloud Free Tier

**Available Resources:**
- 2 AMD-based VMs (1/8 OCPU, 1GB RAM each)
- 4 Arm-based VMs (1/4 OCPU, 4GB RAM each)
- 200GB total block storage
- 10TB monthly bandwidth

**Recommended Configuration:**
- Use 1 Arm-based VM for better performance
- Ubuntu 22.04 LTS ARM64
- Same setup as primary VPS

### 5.2 Standby Configuration

Follow the same setup as primary VPS, but:

1. Use different Redis password
2. Configure Caddy with same domain (for testing)
3. Keep services stopped by default
4. Create deployment scripts for quick activation

```bash
# Create activation script
tee /opt/one-time-link/activate-standby.sh << EOF
#!/bin/bash
sudo systemctl start redis-server
sudo systemctl start one-time-link
sudo systemctl start caddy
echo "Standby services activated"
EOF

chmod +x /opt/one-time-link/activate-standby.sh
```

## 6. Deployment Process

### 6.1 Initial Deployment

1. **Setup DNS records** (allow 24-48 hours for propagation)
2. **Deploy frontend** to Vercel with production environment variables
3. **Setup primary VPS** with Go binary, Redis, and Caddy
4. **Test full flow**: create secret → reveal → already used
5. **Setup standby VPS** and test activation script
6. **Document access credentials** and deployment procedures

### 6.2 Update Deployment

```bash
# On primary VPS
cd /opt/one-time-link
git pull origin main
cd backend
go build -o /opt/one-time-link/api ./cmd/api
sudo systemctl restart one-time-link

# Verify deployment
curl https://api.secret.quorix.io.vn/healthz
```

## 7. Monitoring and Maintenance

### 7.1 Health Checks

Create monitoring script:

```bash
#!/bin/bash
# /opt/one-time-link/health-check.sh

API_URL="https://api.secret.quorix.io.vn/healthz"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $API_URL)

if [ $RESPONSE -eq 200 ]; then
    echo "$(date): API healthy"
else
    echo "$(date): API unhealthy (HTTP $RESPONSE)"
    # Add alerting logic here
fi
```

### 7.2 Log Management

```bash
# Rotate application logs
sudo tee /etc/logrotate.d/one-time-link << EOF
/var/log/caddy/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 caddy caddy
    postrotate
        systemctl reload caddy
    endscript
}
EOF
```

### 7.3 Backup Strategy

**What to backup:**
- Application configuration files
- Deployment scripts
- SSL certificates (if not using Caddy auto-HTTPS)

**What NOT to backup:**
- Redis data (secrets are ephemeral by design)
- Application logs (can be regenerated)

```bash
# Backup script
#!/bin/bash
BACKUP_DIR="/home/deploy/backups/$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR

# Backup configurations
cp /opt/one-time-link/.env $BACKUP_DIR/
cp /etc/caddy/Caddyfile $BACKUP_DIR/
cp /etc/systemd/system/one-time-link.service $BACKUP_DIR/

# Compress backup
tar -czf /home/deploy/backups/config-$(date +%Y%m%d).tar.gz $BACKUP_DIR
rm -rf $BACKUP_DIR
```

## 8. Failover Procedures

### 8.1 Detecting Failure

Monitor these indicators:
- API health endpoint returns non-200 status
- SSH connection to primary VPS fails
- High error rate in application logs
- Redis connectivity issues

### 8.2 Manual Failover Process

1. **Activate standby services:**
   ```bash
   # On Oracle Cloud VPS
   /opt/one-time-link/activate-standby.sh
   ```

2. **Update DNS record:**
   - Change `api.secret.quorix.io.vn` A record to Oracle Cloud IP
   - Wait for DNS propagation (5-15 minutes with low TTL)

3. **Verify failover:**
   ```bash
   # Test API endpoint
   curl https://api.secret.quorix.io.vn/healthz
   
   # Test full flow
   # Create secret → reveal → verify already used
   ```

4. **Monitor and document:**
   - Log failover time and reason
   - Monitor standby performance
   - Plan primary recovery

### 8.3 Failback Process

1. **Diagnose and fix primary VPS issues**
2. **Verify primary VPS health**
3. **Update DNS back to primary IP**
4. **Verify failback successful**
5. **Deactivate standby services to save resources**

## 9. Security Checklist

### 9.1 Server Security

- [ ] SSH key-only authentication
- [ ] Firewall configured (UFW)
- [ ] Non-root user for application
- [ ] Redis password protected
- [ ] Redis bound to localhost only
- [ ] Regular security updates

### 9.2 Application Security

- [ ] HTTPS-only in production
- [ ] CORS restricted to frontend domain
- [ ] Rate limiting enabled
- [ ] Input validation implemented
- [ ] No plaintext logging
- [ ] Security headers configured

### 9.3 Operational Security

- [ ] Environment variables secured
- [ ] Backup encryption (if storing sensitive configs)
- [ ] Access logs monitored
- [ ] Incident response plan documented

## 10. Cost Optimization

### 10.1 Current Costs

- **Vercel**: $0 (Hobby plan sufficient for MVP)
- **Domain**: $0 (existing registration)
- **Primary VPS**: $5-10/month
- **Oracle Cloud**: $0 (Free Tier)
- **Total**: $5-10/month

### 10.2 Scaling Considerations

**Traffic Growth:**
- Monitor Vercel usage (may need Pro plan at $20/month)
- VPS resources may need upgrading
- Consider CDN for static assets

**Feature Growth:**
- Database needs may require PostgreSQL addition
- Background jobs may need separate worker processes
- Monitoring tools may require paid plans

## 11. Success Criteria

The deployment is considered successful when:

- [ ] `secret.quorix.io.vn` loads and functions correctly
- [ ] Create secret flow works end-to-end
- [ ] Reveal gate prevents preview bot consumption
- [ ] First reveal succeeds, second returns "already used"
- [ ] Expired secrets return appropriate error
- [ ] Failover to standby completes within 15 minutes
- [ ] All security headers present
- [ ] Health monitoring functional

This deployment strategy balances cost, learning value, and production readiness for a portfolio project.
