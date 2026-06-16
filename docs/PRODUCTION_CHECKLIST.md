# Production Deployment Checklist

This checklist ensures a safe and successful production deployment of the secret-letter API.

For the current pre-implementation hardening scope and open questions, read [deployment/production-hardening-upgrade-spec.md](deployment/production-hardening-upgrade-spec.md) first.

## Pre-Deployment

### Environment Setup

- [ ] **Production server provisioned**
  - Adequate CPU, memory, and disk space
  - Operating system updated and patched
  - Firewall configured (allow ports 80, 443, and 22 only if needed)
  - Redis is not exposed publicly on `6379`

- [ ] **Redis instance configured**
  - Current MVP baseline uses self-hosted Redis on localhost or a private Compose network
  - Redis AUTH password set if applicable
  - Maxmemory policy reviewed; do not silently evict active secret payloads
  - Persistence configured only if you intentionally want lifecycle metadata retained across restarts
  - If you need remote managed Redis, plan a separate TLS/configuration upgrade first

- [ ] **Domain and DNS configured**
  - Domain registered and pointing to server
  - SSL/TLS certificate obtained (Let's Encrypt, etc.)
  - DNS records configured (A record for API)

### Configuration

- [ ] **Environment variables configured**
  - Copy the deployment-specific template you actually use into your production `.env`
  - Set `APP_SERVICE_NAME` to the production service name
  - Set `APP_HOST` to the intended listen interface
  - Set `APP_PORT` to the intended listen port
  - Set `ALLOWED_ORIGIN` explicitly to your frontend domain; do not rely on localhost defaults
  - Set `TRUSTED_PROXY_CIDRS` explicitly to the reverse proxy or load balancer network
  - Set `REDIS_ADDR` to a private Redis address
  - Set `REDIS_PASSWORD` to Redis AUTH password
  - Set `REDIS_DB` to appropriate database number
  - Set `APP_ENV=production`
  - Generate and store a stable `SECRET_ENCRYPTION_KEY` in a secret manager or production `.env`
  - Configure Redis pool settings based on load
  - Record which deployment mode you are using: single binary + host Caddy, bundled Compose edge, or shared host-level edge

- [ ] **Secrets management**
  - Redis password stored securely (not in version control)
  - `SECRET_ENCRYPTION_KEY` stored securely and backed up
  - `SECRET_ENCRYPTION_KEY` tested across an API restart
  - Environment variables loaded from secure source
  - Secrets rotation policy in place

### Security

- [ ] **Security headers enabled**
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `X-XSS-Protection: 1; mode=block`
  - `Strict-Transport-Security` (HSTS)
  - `Content-Security-Policy`
  - `Referrer-Policy`

- [ ] **HTTPS/TLS configured**
  - Valid SSL/TLS certificate installed
  - HTTP redirects to HTTPS
  - TLS 1.2+ only
  - Strong cipher suites configured

- [ ] **Rate limiting configured**
  - Rate limits appropriate for production load
  - Redis available for rate limiting
  - Rate limit headers included in responses

- [ ] **CORS configured**
  - `ALLOWED_ORIGIN` set to frontend domain only
  - No wildcard (`*`) in production

- [ ] **Trusted proxy boundary verified**
  - Reverse proxy is the component that sets or sanitizes `X-Forwarded-For`
  - `TRUSTED_PROXY_CIDRS` matches the real proxy network only
  - Spoofed forwarded headers from an untrusted source do not change rate-limit identity
  - Run `./scripts/test-trusted-proxy.sh` or `./scripts/test-trusted-proxy.ps1` from the deployment host and archive the output

- [ ] **Runtime hardening completed**
  - Graceful shutdown implemented and verified
  - Redis client connection/read/write timeouts are configured
  - Proxy and API both enforce request body limits
  - The API container runs as a non-root user if you deploy with containers
  - The API container uses a read-only root filesystem with a minimal tmpfs for `/tmp`
  - The API container drops all Linux capabilities and keeps `no-new-privileges`

### Testing

- [ ] **Build successful**
  - Run `go test ./...`
  - Run `gosec ./...`
  - Run `govulncheck ./...`
  - Run `cd frontend/web-app && npm ci && npm test && npm run build && npm audit --omit=dev`
  - All tests pass
  - No security vulnerabilities detected

- [ ] **Integration tests pass**
  - Run integration tests against staging environment
  - Verify all endpoints work correctly
  - Test error scenarios

- [ ] **Load testing completed**
  - Run `./scripts/load-test.sh` or `./scripts/load-test.ps1`
  - Verify performance meets targets (P95 < 100ms)
  - Verify system handles expected load
  - No errors under load

### Monitoring

- [ ] **Logging configured**
  - Log aggregation service configured (optional)
  - Log retention policy in place
  - Sensitive data not logged

- [ ] **Monitoring setup**
  - Health check endpoint monitored (`/healthz`)
  - Readiness endpoint monitored (`/readyz`)
  - Redis connection monitored
  - Error rate monitored
  - Response time monitored

- [ ] **Alerting configured**
  - Alert on health check failures
  - Alert on high error rates (>5%)
  - Alert on slow requests (P95 >200ms)
  - Alert on Redis connection failures

## Deployment

### Pre-Deployment Steps

- [ ] **Backup current version**
  - Backup current binary
  - Backup current configuration
  - Backup Redis volume only if you intentionally rely on Redis persistence for operational recovery
  - Document current version number

- [ ] **Notify stakeholders**
  - Notify team of deployment
  - Schedule maintenance window (if needed)
  - Prepare rollback plan

### Deployment Steps

- [ ] **Upload deployment package**
  - Upload `secret-letter-api-{version}.tar.gz` to server
  - Extract to deployment directory
  - Verify file permissions

- [ ] **Stop current service**
  - Stop running API service
  - Verify service stopped
  - Check for any remaining connections

- [ ] **Deploy new version**
  - Replace binary with new version
  - Update configuration if needed
  - Set executable permissions (`chmod +x`)

- [ ] **Start service**
  - Start API service
  - Verify service started successfully
  - Check logs for errors

- [ ] **Verify deployment**
  - Test health check endpoint
  - Test readiness endpoint
  - Test create secret endpoint
  - Test consume secret endpoint
  - Verify rate limiting works
  - Check error responses

### Post-Deployment Steps

- [ ] **Monitor for issues**
  - Watch logs for errors
  - Monitor error rates
  - Monitor response times
  - Check Redis connections

- [ ] **Smoke tests**
  - Run `./scripts/test-production-smoke.sh` or `./scripts/test-production-smoke.ps1`
  - Create a test secret and save its full link
  - Retrieve test secret status
  - Restart the API service or container before first reveal and verify the existing secret still opens successfully
  - Wait for `/readyz` to recover to `200` before continuing traffic after restart
  - Refresh/open the same link again and verify the consumed state
  - Open a link without the `#fragment` and verify the missing key error
  - Open a link with a malformed fragment and verify the format error without consuming the secret
  - Test hold-to-open on desktop and mobile viewport
  - Send an oversized request and confirm boundary rejection
  - Verify all functionality works

- [ ] **Performance check**
  - Run quick load test
  - Verify response times acceptable
  - Check for any performance degradation

## Post-Deployment

### Verification

- [ ] **Functional verification**
  - All endpoints responding correctly
  - Error handling working as expected
  - Rate limiting enforced
  - CORS working correctly
  - Trusted proxy behavior matches the intended edge topology

- [ ] **Performance verification**
  - Response times within targets
  - No memory leaks
  - No connection pool exhaustion
  - Redis performance acceptable

- [ ] **Security verification**
  - HTTPS working correctly
  - Security headers present
  - Logs contain no plaintext, raw tokens, fragment keys, or unusually long ciphertext values
  - Rate limiting preventing abuse
  - Redis remains private and unreachable from the public internet

### Documentation

- [ ] **Update documentation**
  - Document deployed version
  - Update deployment notes
  - Record any issues encountered
  - Update runbook if needed

- [ ] **Notify stakeholders**
  - Notify team of successful deployment
  - Share deployment notes
  - Update status page (if applicable)

## Rollback Procedure

If issues are detected after deployment:

1. **Stop new service**
   ```bash
   sudo systemctl stop secret-letter-api
   ```

2. **Restore previous version**
   ```bash
   cp /backup/secret-letter-api /usr/local/bin/
   ```

3. **Restore previous configuration**
   ```bash
   cp /backup/.env /opt/secret-letter/
   ```

4. **Start service**
   ```bash
   sudo systemctl start secret-letter-api
   ```

5. **Verify rollback**
   - Test health check
   - Verify functionality
   - Check logs

6. **Investigate issue**
   - Review logs
   - Identify root cause
   - Plan fix for next deployment

## Common Issues

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues and solutions.

## Emergency Contacts

- **On-Call Engineer**: [Contact Info]
- **DevOps Team**: [Contact Info]
- **Redis Support**: [Contact Info]

## Additional Resources

- [Production Deployment Guide](deployment/deployment-guide.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [API Documentation](contracts/public-http-api.md)
- [Monitoring Dashboard](https://monitoring.example.com)

---

**Last Updated**: 2026-04-16  
**Version**: 1.0.0
