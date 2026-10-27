# Backend operations

Production requires an HTTPS `ORIGIN`, `SMTP_HOST`, `SMTP_PORT` (1–65535),
`SMTP_USERNAME`, `SMTP_PASSWORD`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`,
`TURNSTILE_SECRET_KEY`, `TURNSTILE_INVISIBLE_SECRET_KEY`, and a random `PEPPER`
of at least 32 bytes. Startup rejects missing or invalid settings without
printing secret values. `DB_PATH` defaults to `data/app.db`.

Set `DEV=true` explicitly for local development; `DEV=false` means production.
Development defaults `ORIGIN` to `http://localhost:5173` and permits an
unauthenticated local SMTP server, but still requires `SMTP_HOST` and `SMTP_PORT`.

The deployment workflow creates the Podman secret `otp-pepper` with 32 random
bytes (hex encoded) if it does not already exist. It preserves the secret on
subsequent deployments. Manual deployments must provision this secret before
starting `web.service`. Rotating it invalidates outstanding verification codes.
R2 credentials are mounted only into restore and replication containers.

Migration 000002 discards old verification codes. Each account can request one
new code per minute; codes expire after five minutes and allow five attempts.
The frontend must be deployed with the backend: verification now posts
`{"email":"...","code":"..."}` to `/api/auth/verify-account`.

Email messages are constructed and queued during the request. The worker
receives owned message data and retries SMTP delivery up to three times, with
five-second attempt timeouts and one- and two-second retry delays. A full or
stopping queue returns HTTP 503. Shutdown stops producers and drains for up to
ten seconds. Delivery remains best effort: process crashes can lose messages,
exhausted retries are logged, and ambiguous SMTP failures can cause duplicates.
There is no SQLite-backed queue.

Comments and replies use a 15-second shared-cache lifetime with revalidation
and no stale-serving window. Cache purging and its API credentials are no longer
needed. The frontend uses `v=2` on comment/reply reads to avoid old long-lived
cache entries when query strings participate in the CDN cache key. Any custom
Cloudflare rules that ignore query strings or override origin cache headers
must be updated and existing entries purged when deploying this change.

GitHub may link to an existing local account only if that local account is
already verified. An unverified registration is rejected without adding a link,
changing verification status, or creating a session. Its email owner can reset
the password to establish ownership and remove existing sessions before retrying.
Existing OAuth links to unverified accounts are also denied.

## Tests

Run from `backend/`:

```sh
go test ./... -cover
go test -race ./...
```

Tests require no credentials, SMTP server, or external services. Database tests
create temporary SQLite databases and apply the real SQL migrations. They run
sequentially because the application uses a package-level database connection.
Configuration tests set their own environment with `t.Setenv`.
