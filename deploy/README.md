# Deployment

Target: a single VPS running Docker, with host-level nginx for TLS.

## 1. Build the image (local machine)

No registry is used. The image is built locally and copied to the VPS over SSH.

Building on an Apple Silicon Mac produces an arm64 image by default. Most VPS
providers are amd64, so set the platform explicitly:

```bash
docker buildx build --platform linux/amd64 \
  -t authservicebyalien:latest \
  -t authservicebyalien:$(git rev-parse --short HEAD) \
  --load .
```

Use `linux/arm64` if the VPS is ARM. The `--load` flag puts the image in the
local Docker daemon so it can be exported in the next step.

## 2. First deploy

Copy the image to the VPS (from your machine):

```bash
docker save authservicebyalien:latest \
  | gzip \
  | ssh user@vps 'gunzip | docker load'
```

Then on the VPS:

```bash
git clone <repo> /opt/authservicebyalien
cd /opt/authservicebyalien
cp deploy/.env.prod.example .env
# edit .env: AUTH_IMAGE, POSTGRES_PASSWORD, JWT_KEK, SMTP_*
docker compose -f docker-compose.prod.yml up -d
```

`AUTH_IMAGE` must match the tag you loaded. Compose is set to
`pull_policy: never`, so it never contacts a registry.

Migrations run automatically at startup. The container listens on
`127.0.0.1:8082` only.

Bootstrap the first platform admin:

```bash
docker compose -f docker-compose.prod.yml run --rm --entrypoint /app/seed \
  auth-service -email admin@example.com -password 'replace-this'
```

Then sign in at `https://auth.example.com/portal`.

## 3. nginx + TLS

```bash
sudo apt install nginx certbot python3-certbot-nginx
sudo cp deploy/nginx/proxy_params.conf /etc/nginx/snippets/proxy_params.conf
sudo cp deploy/nginx/auth.example.com.conf /etc/nginx/sites-available/auth.example.com
sudo ln -s /etc/nginx/sites-available/auth.example.com /etc/nginx/sites-enabled/
# edit the server_name and the sks backend IP first
sudo certbot --nginx -d auth.example.com
sudo nginx -t && sudo systemctl reload nginx
```

`/auth/v1/*` is restricted to the sks backend IP; only JWKS is public. The
portal is public over TLS but login is rate-limited and password-protected.
Consider adding `auth_basic` to `/portal/` as a second factor against bots.

## 4. Backups

`JWT_KEK` lives in `.env`; the database holds the encrypted signing keys and
all auth data. Back up both.

```bash
# /etc/cron.daily/auth-backup
docker compose -f /opt/authservicebyalien/docker-compose.prod.yml exec -T db \
  sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' \
  | gzip > /var/backups/auth-$(date +%F).sql.gz
```

Store a copy off the VPS. Restoring a database without the matching `JWT_KEK`
will leave the service unable to load signing keys.

## 5. Firewall

Allow only 22 (restricted by source), 80 and 443. The database port is not
published, and `8082` is bound to loopback.

```bash
sudo ufw default deny incoming
sudo ufw allow 80,443/tcp
sudo ufw allow from <your-ip> to any port 22 proto tcp
sudo ufw enable
```

## Deploys after the first

```bash
# on your machine
SHA=$(git rev-parse --short HEAD)
docker buildx build --platform linux/amd64 \
  -t authservicebyalien:$SHA \
  -t authservicebyalien:latest \
  --load .
docker save authservicebyalien:$SHA | gzip | ssh user@vps 'gunzip | docker load'

# on the VPS: set AUTH_IMAGE=authservicebyalien:$SHA in .env, then
docker compose -f docker-compose.prod.yml up -d
```

**Rollback:** set `AUTH_IMAGE` back to a previous SHA (still present on the VPS)
and run `up -d` again.

**Disk:** old images accumulate. Prune weekly:

```bash
docker image prune -a --filter until=168h
```

If you later move to a registry, remove `pull_policy: never`, set `AUTH_IMAGE`
to the registry reference, and `docker compose pull` before `up -d`.

## Notes

- `TRUSTED_PROXIES` is set to the compose subnet plus loopback because nginx
  reaches the container through the published loopback port.
- `POSTGRES_PASSWORD` is interpolated into `DATABASE_URL`, so keep it
  URL-safe (letters/digits) or percent-encode reserved characters. Compose also
  expands `$` inside `.env`; escape it as `$$` if your password contains one.
- `sslmode=disable` is used for the app-to-DB hop inside the compose network.
  If you move to a managed database, enable TLS and update `DATABASE_URL`.
- The portal cookie is `Secure` because `APP_ENV=production`.
- Rotating `JWT_KEK` requires re-encrypting `signing_keys`; there is no
  built-in re-wrap command yet.
