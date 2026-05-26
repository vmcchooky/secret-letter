# Production Deployment Checklist

This checklist ensures a safe and successful production deployment of the one-time-link API.

## Pre-Deployment

### Environment Setup

- [ ] **Production server provisioned**
  - Adequate CPU, memory, and disk space
  - Operating system updated and patched
  - Firewall configured (allow ports 80, 443, 6379)

- [ ] **Redis instance configured**
  - Managed Redis service (AWS ElastiCache, Azure Cache, etc.) OR
  - Self-hosted Redis with persistence enabled
  - Redis AUTH password set
  - TLS/SSL enabled for Redis connections
  - Maxmemory policy configured (recommend: `allkeys-lru`)
  - Persistence configured (AOF or RDB)

- [ ] **Domain and DNS configured**
  - Domain registered and pointing to server
  - SSL/TLS certificate obtained (Let's Encrypt, etc.)
  - DNS records configured (A record for API)

### Configuration

- [ ] **Environment variables configured**
  - Copy `.env.production` to `.env`
  - Set `SERVICE_NAME` to production service name
  - Set `HOST` to `0.0.0.0` (listen on all interfaces)
  - Set `PORT` to `8080` (or your preferred port)
  - Set `ALLOWED_ORIGIN` to your frontend domain
  - Set `REDIS_ADDR` to production Redis address
  - Set `REDIS_PASSWORD` to Redis AUTH password
  - Set `REDIS_DB` to appropriate database number
  - Set `APP_ENV=production`
  - Generate and store a stable `SECRET_ENCRYPTION_KEY` in a secret manager or production `.env`
  - Set `TRUSTED_PROXY_CIDRS` to only the reverse proxy/load balancer network
  - Configure Redis pool settings based on load

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

### Testing

- [ ] **Build successful**
  - Run `go test ./...`
  - Run `gosec ./...`
  - Run `govulncheck ./...`
  - Run `cd frontend/web-app && npm ci && npm run build && npm audit --omit=dev`
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
  - Backup Redis volume/data directory before deploy
  - Document current version number

- [ ] **Notify stakeholders**
  - Notify team of deployment
  - Schedule maintenance window (if needed)
  - Prepare rollback plan

### Deployment Steps

- [ ] **Upload deployment package**
  - Upload `one-time-link-api-{version}.tar.gz` to server
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
  - Create a test secret
  - Retrieve test secret status
  - Open test secret once
  - Refresh/open the same link again and verify the consumed state
  - Open a link without the `#fragment` and verify the missing key error
  - Test hold-to-open on desktop and mobile viewport
  - Verify all functionality works
  - Restart the API service/container, create another secret, and open it successfully

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
   sudo systemctl stop one-time-link-api
   ```

2. **Restore previous version**
   ```bash
   cp /backup/one-time-link-api /usr/local/bin/
   ```

3. **Restore previous configuration**
   ```bash
   cp /backup/.env /opt/one-time-link/
   ```

4. **Start service**
   ```bash
   sudo systemctl start one-time-link-api
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
