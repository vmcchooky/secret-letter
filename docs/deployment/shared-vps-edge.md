# secret-letter On The Shared Quorix VPS

This is the **primary production reference path** for the current hardening sprint and first public go-live.

Use this when `secret.quorix.io.vn` and `api.secret.quorix.io.vn` are served by the host-level Caddy edge.

## Backend

```sh
cd /opt/secret-letter
cp deploy/prod/.env.vps-edge.example deploy/prod/.env
nano deploy/prod/.env

docker compose \
  -f deploy/prod/docker-compose.yml \
  -f deploy/prod/docker-compose.vps-edge.yml \
  up -d --build redis api
```

The API listens on `127.0.0.1:18080`; Redis stays private in the Compose network.

The base production Compose file now keeps the bundled `caddy` service behind the opt-in `bundled-edge` profile. The shared-edge override keeps that service disabled so the host-level Caddy edge is the only public entrypoint.

The API container now also runs with a read-only root filesystem, a minimal `/tmp` tmpfs, dropped Linux capabilities, and `no-new-privileges`.

Recommended rate-limit values for the shared VPS are configured through `deploy/prod/.env`:

```sh
APP_ENV=production
RATE_LIMIT_ENABLED=true
RATE_LIMIT_WINDOW_SECONDS=3600
RATE_LIMIT_CREATE_PER_WINDOW=120
RATE_LIMIT_CONSUME_PER_WINDOW=240
RATE_LIMIT_STATUS_PER_WINDOW=600
RATE_LIMIT_REVEAL_SESSION_PER_WINDOW=240
TRUSTED_PROXY_CIDRS=172.16.0.0/12
```

## Host Caddy Edge Boundary

Mirror the API boundary rules at the host-level Caddy edge:

```caddyfile
api.secret.quorix.io.vn {
    tls {
        protocols tls1.2 tls1.3
    }

    route {
        request_body {
            max_size 32KB
        }

        reverse_proxy 127.0.0.1:18080
    }

    encode zstd gzip

    header {
        X-Content-Type-Options nosniff
        X-Frame-Options DENY
        Referrer-Policy no-referrer
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
}
```

Use Caddy `2.10+` on the host if you rely on the `request_body` directive for proxy-layer size enforcement.

Keep the forwarded-header trust boundary tight:

- Let the edge overwrite `X-Forwarded-For`, `X-Forwarded-Proto`, and `X-Forwarded-Host`; do not pass client-supplied values through unchanged.
- Keep `TRUSTED_PROXY_CIDRS` scoped to the final hop that reaches the API container, not the public internet.
- If another proxy or CDN sits in front of the host Caddy edge, configure that trust relationship there first before expanding the API trust list.

## Frontend

```sh
cd /opt/secret-letter/frontend/web-app
cp ../../deploy/prod/frontend.env.vps-edge.example .env.production
npm ci
npm run build

sudo install -d -m 755 /var/www/secret-letter/frontend/dist
sudo rsync -a --delete dist/ /var/www/secret-letter/frontend/dist/
```

The host Caddy edge serves `/var/www/secret-letter/frontend/dist` for `secret.quorix.io.vn` and uses SPA fallback for app routes, including root-level short secret links such as `/:secretId`.

## Verify

```sh
curl -I http://127.0.0.1:18080/healthz
curl -I https://api.secret.quorix.io.vn/healthz
curl -I https://secret.quorix.io.vn/
```

Confirm oversized requests are rejected at the proxy layer:

```sh
python3 - <<'PY' > /tmp/secret-letter-oversized.json
import json

print(json.dumps({
    "ciphertext": "A" * (40 * 1024),
    "nonce": "MTIzNDU2Nzg5MDEy",
    "algorithm": "AES-GCM",
    "ttlSeconds": 3600,
}))
PY

curl -s -o /dev/null -w "%{http_code}\n" \
  -H 'Content-Type: application/json' \
  --data-binary @/tmp/secret-letter-oversized.json \
  https://api.secret.quorix.io.vn/api/secrets
# expect: 413
```

From the repo root on the VPS, you can also run the repeatable verification scripts:

```sh
./scripts/test-trusted-proxy.sh
./scripts/test-production-smoke.sh
```

To verify the API runtime hardening directly:

```sh
docker inspect secret-letter-api --format '{{.Config.User}} {{.HostConfig.ReadonlyRootfs}} {{json .HostConfig.CapDrop}}'
# expect: app true ["ALL"]
```

Then run the product smoke test:

1. Create a secret.
2. Open the generated `/:secretId#key` link once.
3. Open the same link again and confirm the consumed state.
4. Open a link without the fragment and confirm it does not consume the secret.

## Notes

- Do not start the Compose `caddy` service on the shared VPS.
- Keep `SECRET_ENCRYPTION_KEY` stable across deploys and restarts.
- Keep `ALLOWED_ORIGIN=https://secret.quorix.io.vn`; do not use wildcard CORS in production.
