# Troubleshooting Guide

This guide helps diagnose and resolve common issues with the one-time-link API.

## Table of Contents

- [Service Won't Start](#service-wont-start)
- [Redis Connection Issues](#redis-connection-issues)
- [Rate Limiting Issues](#rate-limiting-issues)
- [Performance Issues](#performance-issues)
- [Error Responses](#error-responses)
- [CORS Issues](#cors-issues)
- [Logging Issues](#logging-issues)

---

## Service Won't Start

### Symptom
Service fails to start or exits immediately after starting.

### Possible Causes

#### 1. Port Already in Use

**Check:**
```bash
# Linux/Mac
lsof -i :8080

# Windows
netstat -ano | findstr :8080
```

**Solution:**
- Stop the conflicting service
- Change the port in `.env` file
- Update firewall rules if needed

#### 2. Missing Environment Variables

**Check:**
```bash
# Verify .env file exists
ls -la .env

# Check environment variables
env | grep -E "SERVICE_NAME|REDIS_ADDR|PORT"
```

**Solution:**
- Copy `.env.production` to `.env`
- Fill in all required variables
- Restart service

#### 3. Redis Connection Failed

**Check:**
```bash
# Test Redis connection
redis-cli -h localhost -p 6379 ping
```

**Solution:**
- Verify Redis is running
- Check Redis address in `.env`
- Verify Redis password (if set)
- Check network connectivity

#### 4. Permission Issues

**Check:**
```bash
# Check binary permissions
ls -l one-time-link-api

# Check file ownership
ls -l .env
```

**Solution:**
```bash
# Make binary executable
chmod +x one-time-link-api

# Fix ownership
chown user:group one-time-link-api .env
```

---

## Redis Connection Issues

### Symptom
API returns 500 errors, logs show Redis connection failures.

### Diagnosis

**Check logs:**
```bash
# Look for Redis errors
grep -i "redis" /var/log/one-time-link/api.log
```

**Test Redis connection:**
```bash
# Direct connection test
redis-cli -h <redis-host> -p <redis-port> -a <password> ping
```

### Solutions

#### 1. Redis Not Running

**Check:**
```bash
# Linux
systemctl status redis

# Docker
docker ps | grep redis
```

**Solution:**
```bash
# Start Redis
systemctl start redis

# Or start Docker container
docker start redis
```

#### 2. Wrong Redis Address

**Check `.env`:**
```bash
cat .env | grep REDIS_ADDR
```

**Solution:**
- Update `REDIS_ADDR` to correct address
- Format: `host:port` (e.g., `localhost:6379`)
- Restart API service

#### 3. Authentication Failed

**Check:**
```bash
# Test with password
redis-cli -h <host> -p <port> -a <password> ping
```

**Solution:**
- Verify `REDIS_PASSWORD` in `.env`
- Check Redis `requirepass` configuration
- Restart API service

#### 4. Connection Pool Exhausted

**Check logs for:**
```
redis: connection pool: failed to dial after X attempts
```

**Solution:**
- Increase `REDIS_POOL_SIZE` in `.env`
- Increase `REDIS_MIN_IDLE` in `.env`
- Check for connection leaks
- Monitor Redis connections

---

## Rate Limiting Issues

### Symptom
Users getting 429 (Too Many Requests) errors unexpectedly.

### Diagnosis

**Check rate limit headers:**
```bash
curl -I http://localhost:8080/api/secrets
# Look for X-RateLimit-* headers
```

**Check Redis keys:**
```bash
redis-cli keys "ratelimit:*"
redis-cli get "ratelimit:create_secret:192.168.1.1"
```

### Solutions

#### 1. Rate Limits Too Strict

**Current limits:**
- Create secret: 120/hour per IP
- Consume secret: 240/hour per IP
- Status check: 600/hour per IP
- Reveal session: 240/hour per IP

**Solution:**
- Adjust limits in `backend/internal/httpapi/middleware.go`
- Rebuild and redeploy
- Or implement per-user rate limiting

#### 2. IP Address Detection Issues

**Check logs for IP addresses:**
```bash
grep "ip_hash" /var/log/one-time-link/api.log
```

**Solution:**
- Verify `X-Forwarded-For` header is set by load balancer
- Check `X-Real-IP` header
- Update `getClientIP()` function if needed

#### 3. Redis Keys Not Expiring

**Check TTL:**
```bash
redis-cli ttl "ratelimit:create_secret:192.168.1.1"
```

**Solution:**
- Verify Redis EXPIRE command is working
- Check Redis maxmemory-policy
- Manually delete keys if needed: `redis-cli del "ratelimit:*"`

---

## Performance Issues

### Symptom
Slow response times, high latency, timeouts.

### Diagnosis

**Check slow request logs:**
```bash
grep "slow_request" /var/log/one-time-link/api.log | tail -20
```

**Run load test:**
```bash
./scripts/load-test.sh --concurrent 10 --requests 100
```

**Check system resources:**
```bash
# CPU usage
top

# Memory usage
free -h

# Disk I/O
iostat -x 1
```

### Solutions

#### 1. High Redis Latency

**Check Redis performance:**
```bash
redis-cli --latency
redis-cli --latency-history
```

**Solution:**
- Check Redis memory usage: `redis-cli info memory`
- Enable Redis persistence (AOF or RDB)
- Use Redis cluster for high availability
- Increase Redis memory limit

#### 2. Connection Pool Issues

**Check logs for:**
```
connection pool: failed to dial
```

**Solution:**
- Increase `REDIS_POOL_SIZE`
- Increase `REDIS_MIN_IDLE`
- Check for connection leaks
- Monitor active connections

#### 3. High CPU Usage

**Check:**
```bash
top -p $(pgrep one-time-link-api)
```

**Solution:**
- Check for infinite loops in code
- Profile application: `go tool pprof`
- Optimize hot paths
- Scale horizontally (add more instances)

#### 4. Memory Leaks

**Check:**
```bash
# Monitor memory over time
watch -n 5 'ps aux | grep one-time-link-api'
```

**Solution:**
- Profile memory usage: `go tool pprof -alloc_space`
- Check for goroutine leaks
- Restart service periodically (temporary fix)
- Fix memory leak in code

---

## Error Responses

### Symptom
API returning unexpected error responses.

### Common Errors

#### 1. 400 Bad Request - Validation Failed

**Response:**
```json
{
  "error": "validation_failed",
  "message": "Request validation failed",
  "details": {
    "validation_errors": [...]
  }
}
```

**Solution:**
- Check request payload format
- Verify all required fields present
- Check field values (algorithm, TTL, etc.)
- Review validation error details

#### 2. 410 Gone - Already Consumed

**Response:**
```json
{
  "error": "SECRET_CONSUMED",
  "message": "This secret has already vanished."
}
```

**Solution:**
- Secret can only be consumed once
- Check if secret has expired (TTL)
- Create a new secret if needed

#### 3. 429 Too Many Requests

**Response:**
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please try again later.",
  "details": {
    "retry_after": 3600
  }
}
```

**Solution:**
- Wait for rate limit to reset (check `Retry-After` header)
- Reduce request frequency
- Contact admin to adjust rate limits

#### 4. 500 Internal Server Error

**Response:**
```json
{
  "error": "internal_error",
  "message": "Failed to create secret"
}
```

**Solution:**
- Check server logs for details
- Verify Redis connection
- Check system resources
- Contact support if issue persists

---

## CORS Issues

### Symptom
Frontend cannot access API, browser shows CORS errors.

### Diagnosis

**Check browser console:**
```
Access to fetch at 'http://api.example.com' from origin 'http://frontend.example.com' 
has been blocked by CORS policy
```

**Check CORS headers:**
```bash
curl -H "Origin: http://frontend.example.com" \
     -H "Access-Control-Request-Method: POST" \
     -H "Access-Control-Request-Headers: Content-Type" \
     -X OPTIONS \
     http://api.example.com/api/secrets
```

### Solutions

#### 1. Wrong ALLOWED_ORIGIN

**Check `.env`:**
```bash
cat .env | grep ALLOWED_ORIGIN
```

**Solution:**
- Set `ALLOWED_ORIGIN` to your frontend domain
- Format: `https://frontend.example.com` (no trailing slash)
- Restart API service

#### 2. Missing CORS Headers

**Check response headers:**
```bash
curl -I http://api.example.com/api/secrets
```

**Solution:**
- Verify `Access-Control-Allow-Origin` header present
- Verify `Access-Control-Allow-Methods` header present
- Check middleware configuration

#### 3. Preflight Request Failing

**Test OPTIONS request:**
```bash
curl -X OPTIONS http://api.example.com/api/secrets
```

**Solution:**
- Verify OPTIONS requests return 204 No Content
- Check CORS middleware handles OPTIONS
- Verify no authentication required for OPTIONS

---

## Logging Issues

### Symptom
Logs not appearing, logs missing information, log format issues.

### Diagnosis

**Check log output:**
```bash
# Follow logs
tail -f /var/log/one-time-link/api.log

# Check log permissions
ls -l /var/log/one-time-link/
```

### Solutions

#### 1. Logs Not Appearing

**Check:**
- Verify log directory exists
- Check write permissions
- Check disk space: `df -h`

**Solution:**
```bash
# Create log directory
mkdir -p /var/log/one-time-link

# Fix permissions
chown user:group /var/log/one-time-link
chmod 755 /var/log/one-time-link
```

#### 2. Log Format Issues

**Check log format:**
```bash
cat /var/log/one-time-link/api.log | head -5
```

**Solution:**
- Logs should be JSON format
- Verify structured logging is enabled
- Check for log parsing errors

#### 3. Too Many Logs

**Check log size:**
```bash
du -sh /var/log/one-time-link/
```

**Solution:**
- Implement log rotation
- Reduce log verbosity
- Archive old logs
- Use log aggregation service

---

## Getting Help

If you cannot resolve the issue:

1. **Collect information:**
   - Error messages from logs
   - Steps to reproduce
   - Environment details (OS, Go version, Redis version)
   - Configuration (sanitized, no passwords)

2. **Check documentation:**
   - [API Documentation](contracts/public-http-api.md)
   - [Deployment Guide](deployment/deployment-guide.md)
   - [Production Checklist](PRODUCTION_CHECKLIST.md)

3. **Contact support:**
   - Create GitHub issue with details
   - Contact on-call engineer
   - Email support team

---

**Last Updated**: 2026-04-16  
**Version**: 1.0.0
