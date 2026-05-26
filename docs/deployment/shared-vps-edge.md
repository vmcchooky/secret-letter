# secret-letter On The Shared Quorix VPS

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

Then run the product smoke test:

1. Create a secret.
2. Open the generated `/:secretId#key` link once.
3. Open the same link again and confirm the consumed state.
4. Open a link without the fragment and confirm it does not consume the secret.

## Notes

- Do not start the Compose `caddy` service on the shared VPS.
- Keep `SECRET_ENCRYPTION_KEY` stable across deploys and restarts.
- Keep `ALLOWED_ORIGIN=https://secret.quorix.io.vn`; do not use wildcard CORS in production.
