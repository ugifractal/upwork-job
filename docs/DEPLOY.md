# Deploy Runbook: upwork-job

Go API for saving Upwork jobs, deployed to the home server `drx` via [Kamal](https://kamal-deploy.org/).

- **Domain:** `https://jobhist.duruxignition.com`
- **Server:** `drx.catfish-algol.ts.net` (Tailscale), Linux amd64
- **Registry:** `ghcr.io/ugifractal/upwork-job`
- **Storage:** SQLite at `/var/lib/upwork-job/jobs.db` (bind mount)

---

## Prerequisites

From the Mac where you normally deploy (the machine that already runs other Kamal apps):

```bash
kamal version                                    # Kamal is installed
ssh sugiarto@drx.catfish-algol.ts.net 'docker ps' # SSH + Docker reachable via Tailscale
```

> Note: the app fails fast (crash-loops) if `API_KEY` or any `SMTP_*` value is
> missing. Make sure all secrets in Phase 2 are complete before deploying.

## Phase 1 — One-time server prep

```bash
ssh sugiarto@drx.catfish-algol.ts.net \
  'sudo mkdir -p /var/lib/upwork-job && sudo chown 65532:65532 /var/lib/upwork-job'
```

The container runs as the distroless `nonroot` user (uid 65532), so the host
directory must be writable by that uid.

## Phase 2 — Secrets

Edit `.kamal/secrets` in the repo (besides `GHCR_TOKEN`, add):

```
API_KEY=<strong-random-key>
SMTP_HOST=smtp.zoho.com
SMTP_PORT=587
SMTP_USER=admin@duruxignition.com
SMTP_PASS=<your-zoho-password>
SMTP_FROM=admin@duruxignition.com
NOTIFY_EMAIL=ugidmtest@gmail.com
```

Verify it stays out of git: `git status` should NOT list `.kamal/secrets`
(it is gitignored).

## Phase 3 — Deploy

```bash
kamal env push                                # resolve secrets → env file on server
kamal deploy                                  # build amd64 → push to GHCR → deploy via kamal-proxy
kamal app exec './upwork-job -migrate=up'     # create schema (before traffic is enabled)
```

## Phase 4 — Local healthcheck (before going public)

```bash
ssh sugiarto@drx.catfish-algol.ts.net \
  'curl -s -H "Host: jobhist.duruxignition.com" http://localhost/health'
# → {"status":"ok"}

kamal app logs -n 50                          # no fatal errors
```

## Phase 5 — Cloudflare

1. Zero Trust → Tunnels → **Public Hostname** → add `jobhist.duruxignition.com`
2. Service:
   - cloudflared on the **server** → `http://localhost:80`
   - cloudflared on the **Mac** → `http://drx.catfish-algol.ts.net:80`
3. The DNS record for `jobhist` is created automatically.
4. If needed, set SSL/TLS encryption mode to **Full** in the Cloudflare dashboard.

## Phase 6 — End-to-end verification

```bash
curl https://jobhist.duruxignition.com/health

curl -X POST https://jobhist.duruxignition.com/api/jobs \
  -H "Content-Type: application/json" -H "X-API-Key: <KEY>" \
  -d '{"upwork_job_id":"16823456789","title":"Go Developer Needed","summary":"Build REST API in Go","link":"https://www.upwork.com/jobs/~01abc123"}'

# 1st POST       → 201 {"status":"created"} + email to NOTIFY_EMAIL
# 2nd POST       → 200 {"status":"skipped"}
# DB on server   → /var/lib/upwork-job/jobs.db grows
```

## Updating

```bash
kamal env push   # if secrets changed
kamal deploy
kamal app exec './upwork-job -migrate=up'   # idempotent (IF NOT EXISTS)
```

## Rollback

```bash
kamal rollback <version>   # rolling back container; SQLite data is untouched
```

## Changing the domain

1. Update `proxy.hosts` in `config/deploy.yml`.
2. Update the Cloudflare Public Hostname (see Phase 5).