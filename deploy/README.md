# Deployment Configuration

Configuration files cho local development và production deployment.

## Current Status

**Milestone:** 4/7 complete (57%)  
**Status:** Production-ready, awaiting deployment  
**Next:** Production deployment (Milestone 5)

## Local Development

### Redis Container

`local/docker-compose.yml` - Minimal Redis container cho local development

**Start Redis:**
```bash
docker compose -f deploy/local/docker-compose.yml up -d
```

**Stop Redis:**
```bash
docker compose -f deploy/local/docker-compose.yml down
```

**Check Status:**
```bash
docker compose -f deploy/local/docker-compose.yml ps
```

**Configuration:**
- Port: 6379
- No password (local only)
- No persistence (development only)

## Production Deployment

### Reverse Proxy

`prod/Caddyfile` - Caddy reverse proxy configuration cho API domain
`prod/docker-compose.vps-edge.yml` - Override for the shared Quorix VPS edge; publishes the API on `127.0.0.1:18080` and disables the bundled Caddy service.

**Features:**
- Automatic HTTPS với Let's Encrypt
- Reverse proxy to Go API
- Security headers
- Access logging

**Usage:**
```bash
# Install Caddy
# See: https://caddyserver.com/docs/install

# Run Caddy
caddy run --config prod/Caddyfile
```

### Environment Configuration

`prod/.env.example` - Sample environment variables cho production

**Required Variables:**
```bash
APP_SERVICE_NAME=one-time-link-api
APP_ENV=production
APP_HOST=0.0.0.0
APP_PORT=8080
ALLOWED_ORIGIN=https://your-frontend.com
TRUSTED_PROXY_CIDRS=172.16.0.0/12
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=your-secure-password
REDIS_DB=0
REDIS_POOL_SIZE=50
REDIS_MIN_IDLE=10
REDIS_MAX_RETRIES=3
SECRET_ENCRYPTION_KEY=base64url-encoded-32-byte-key
RATE_LIMIT_ENABLED=true
RATE_LIMIT_WINDOW_SECONDS=3600
RATE_LIMIT_CREATE_PER_WINDOW=120
RATE_LIMIT_CONSUME_PER_WINDOW=240
RATE_LIMIT_STATUS_PER_WINDOW=600
RATE_LIMIT_REVEAL_SESSION_PER_WINDOW=240
```

Generate `SECRET_ENCRYPTION_KEY` once and keep it stable across API restarts:

```bash
openssl rand -base64 32 | tr '+/' '-_' | tr -d '='
```

Set `TRUSTED_PROXY_CIDRS` to the network range used by your reverse proxy. For the bundled Docker Compose setup, `172.16.0.0/12` covers the Docker bridge network; do not use `0.0.0.0/0`.

**Setup:**
```bash
# Copy template
cp prod/.env.example .env

# Edit with your values
nano .env
```

## Production Deployment Guide

For complete deployment instructions, see:
- [Production Checklist](../docs/PRODUCTION_CHECKLIST.md)
- [Deployment Guide](../docs/deployment/deployment-guide.md)
- [Troubleshooting](../docs/TROUBLESHOOTING.md)

### Quick Deployment Steps

1. **Build production binary:**
   ```bash
   ./scripts/build-production.sh
   ```

2. **Setup VPS:**
   - Install Go, Redis, Caddy
   - Configure firewall (ports 80, 443, 6379)
   - Setup systemd service

3. **Configure environment:**
   ```bash
   cp deploy/prod/.env.example /opt/one-time-link/.env
   # Edit .env with production values
   ```

4. **Deploy binary:**
   ```bash
   # Upload and extract deployment package
   tar -xzf one-time-link-api-{version}.tar.gz
   cp one-time-link-api-linux-amd64 /usr/local/bin/
   ```

5. **Start services:**
   ```bash
   systemctl start redis
   systemctl start caddy
   systemctl start one-time-link-api
   ```

6. **Verify:**
   ```bash
   curl https://api.your-domain.com/healthz
   curl https://api.your-domain.com/readyz
   ```

## Architecture

### Local Development
```
Frontend (localhost:5173)
    ↓
Backend (localhost:8080)
    ↓
Redis (localhost:6379)
```

### Production
```
Users
    ↓
Frontend (Vercel)
    ↓
Caddy (HTTPS, reverse proxy)
    ↓
Go API (localhost:8080)
    ↓
Redis (localhost:6379 with AUTH)
```

## Security Considerations

### Local Development
- ✅ No password required (local only)
- ✅ No HTTPS (local only)
- ✅ CORS allows localhost:5173

### Production
- ✅ Redis AUTH password required
- ✅ HTTPS only (Caddy + Let's Encrypt)
- ✅ CORS restricted to frontend domain
- ✅ Security headers enabled
- ✅ Rate limiting enabled
- ✅ Firewall configured

## Monitoring

### Health Check
```bash
# Local
curl http://localhost:8080/healthz

# Production
curl https://api.your-domain.com/healthz
```

### Redis Status
```bash
# Local
redis-cli ping

# Production
redis-cli -a your-password ping
```

### Logs
```bash
# Backend logs
journalctl -u one-time-link-api -f

# Caddy logs
journalctl -u caddy -f

# Redis logs
journalctl -u redis -f
```

## Troubleshooting

See [TROUBLESHOOTING.md](../docs/TROUBLESHOOTING.md) for common issues and solutions.

**Common Issues:**
- Service won't start → Check Redis connection
- CORS errors → Verify ALLOWED_ORIGIN
- Rate limiting → Check Redis keys
- Slow performance → Run load test

## Future Enhancements

- [x] Docker Compose for production
- [x] Shared VPS edge override
- [ ] Kubernetes manifests
- [ ] Terraform configurations
- [ ] CI/CD pipeline
- [ ] Monitoring stack (Prometheus + Grafana)
- [ ] Backup automation

---

**Note:** This folder is intentionally minimal. It provides just enough structure for local development và low-cost VPS deployment without locking into a heavier platform too early. As the project grows, more sophisticated deployment options can be added.
