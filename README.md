# Muster

Group hiking outings with built-in carpool coordination. Hosts schedule hikes, hikers request to join as drivers or riders, and seat math is enforced at acceptance — nobody gets stranded in the parking lot.

**Status: v0 backend complete.** REST API with cookie-based auth, capacity-checked membership, and a CI-gated test suite. Frontend (React) in progress.

## Stack

- **Go** (net/http, stdlib mux) — no framework
- **PostgreSQL** on [Neon](https://neon.tech), via `pgx`; migrations with `goose`
- **[go-toolkit/v2](https://github.com/diagnosis/go-toolkit)** — shared error vocabulary, JSON envelope, structured logging, middleware (correlation IDs, rate limiting, auth), Argon2id + JWT
- **Playwright** (TypeScript) — 31-test e2e suite against the real HTTP surface
- **GitHub Actions** — lint + unit tests, then the e2e suite against a per-run disposable Neon branch

## Architecture

```
cmd/muster         wires everything
internal/api       HTTP handlers → hiker.Service, outing.Service (never postgres)
internal/hiker     auth + profile domain: rules, Storage interface
internal/outing    outings, join requests, capacity: rules, Storage interface
internal/postgres  implements both Storage interfaces; owns all error translation
internal/config    env-driven config, loud validation
```

Domain packages import only the toolkit — never each other (IDs, not imports). Services define the `Storage` interfaces they consume; postgres implements them. Business rules live in services and are unit-tested against in-memory fakes; SQL is exercised by the e2e suite.

## API surface (v0)

| Area | Endpoints |
|---|---|
| Health | `GET /api/health` |
| Auth | signup · login · refresh (rotating) · logout · `GET /api/auth/me` |
| Profile | `PATCH /api/me/profile` · `GET /api/hikers/{id}` (public card, no email) |
| Outings | list upcoming · `GET /api/outings/{id}` (public, viewer-aware) · create · `PATCH` · cancel |
| Membership | request join · withdraw · accept · decline · remove member · host inbox · `GET /api/me/outings` |

Design notes worth knowing:

- **Reads are public, writes are authed.** `GET /api/outings/{id}` works anonymously; logged-in viewers additionally get `my_request` (their own standing on the outing).
- **Capacity gates acceptance, not asking.** Anyone can request; the host's accept fails with `409` if the person (plus guests) doesn't fit — checked atomically in SQL, so concurrent accepts can't oversell.
- **Seat math is derived, never stored:** `people_count`, `seat_capacity`, `seats_short`, `spots_left` are computed per read. Hosts may shrink capacity below commitments; the shortage surfaces in the detail view and a human resolves it.
- **Cookie sessions** (HttpOnly access + rotating refresh), one session per platform.
- Responses use a uniform envelope: `{data, correlation_id, timestamp}` / `{error: {...}}`.

## Running locally

Requirements: Go 1.24+, a Postgres URL (Neon works), `goose`, Node 20+ (for e2e).

```bash
# backend/.env
DATABASE_URL=postgres://...
JWT_ACCESS_SECRET=...        # openssl rand -base64 48
JWT_REFRESH_SECRET=...       # must differ from access secret
APP_ENV=dev
APP_PORT=8088
```

```bash
cd backend
make check        # gofmt + golangci-lint + build + unit tests
make run          # boots on :8088
make e2e          # Playwright suite (server must be running)
make e2e-branch   # fresh 24h-TTL Neon branch + migrations; prints its DATABASE_URL
```

Optional env: `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` (defaults 10/20), JWT expiries, DB pool tuning — see `internal/config`.

## Testing

- **Unit** (`go test ./...`): 43 tests over the service layer against in-memory fakes — the state machine, capacity table, and validation rules.
- **e2e** (`backend/e2e`): 31 Playwright tests speaking real HTTP to a real database. Per-test actors with unique emails — no shared state, no cleanup, parallel-safe by construction. Conventions live in [`backend/e2e/TEST-DESIGN.md`](backend/e2e/TEST-DESIGN.md).
- **CI**: every PR to `main` runs lint + unit tests, then creates a disposable Neon branch, migrates it, boots the server, runs the full e2e suite, and deletes the branch. Green gates the merge.

## Roadmap

v1 candidates (in rough order): editable join requests, host notifications, outing group chat, post-hike contact exchange, trail-conditions/wildlife notes, mobile header-auth. See commit history for design rulings — most carry their reasoning.
