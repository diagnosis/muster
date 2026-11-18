[![CI](https://github.com/diagnosis/muster/actions/workflows/ci.yml/badge.svg)](https://github.com/diagnosis/muster/actions/workflows/ci.yml)
[![deploy.yml](https://github.com/diagnosis/muster/actions/workflows/deploy.yml/badge.svg)](https://github.com/diagnosis/muster/actions/workflows/deploy.yml)
# Muster

Group hiking outings with built-in carpool coordination. Hosts schedule hikes, hikers ask to join as drivers or riders, and seat math is enforced when the host accepts, so nobody is stranded in the parking lot.

**Status: v1 live at [muster.safadev.app](https://muster.safadev.app).** A Go REST API, a React web app, and three test suites that gate every pull request. **Next:** v2 with a mobile app.

## What it does

- **Outings:** create, edit and cancel a hike with a meeting point, group size, pace, terrain and cost per seat. The browse list is public and paginated.
- **Joining:** hikers ask to join as a rider or a driver, with guests and seats offered. The host accepts, declines, removes or restores. Capacity is checked at acceptance.
- **Live pages:** the outing page, browse and My outings update over server-sent events when a roster or an outing changes.
- **Talking:** a discussion thread on each outing, a group chat for its members, and direct messages that the other hiker has to accept.
- **Notifications:** an in-app bell and email for the events that concern you.
- **Accounts:** email verification, password or emailed-code login, password reset, a profile, and a public hiker card with hosted and attended counts.

## Stack

- **Go 1.26** (`net/http`, standard-library mux), no framework
- **PostgreSQL 16** via `pgx`; migrations with `goose`
- **[go-toolkit/v3](https://github.com/diagnosis/go-toolkit)**: shared error vocabulary, JSON envelope, structured logging, middleware (correlation IDs, rate limiting, auth), Argon2id and JWT
- **React 19, TypeScript, Vite**, TanStack Router and TanStack Query, CSS Modules
- **Server-sent events** for live updates; **Resend** for email
- **Playwright** (TypeScript) for an API suite and a browser suite
- **GitHub Actions** for CI and deploy

## Architecture

```
backend/
  cmd/muster             wires everything
  internal/api           HTTP handlers and middleware; calls services, never postgres
  internal/hiker         accounts, sessions, profile, public hiker card
  internal/outing        outings, join requests, capacity, comments
  internal/message       outing chat and direct messages
  internal/notification  in-app notifications and the email dispatcher
  internal/events        the SSE hub
  internal/authtoken     one-time tokens: email verification, login codes, password reset
  internal/email         the mail sender
  internal/postgres      implements every Storage interface; owns error translation
  internal/config        env-driven config, loud validation
  migrations             goose SQL migrations
  e2e                    Playwright API suite
frontend/
  src                    routes, queries, components
  e2e                    Playwright browser suite
```

Domain packages import only the toolkit, never each other (IDs, not imports). Each service defines the `Storage` interface it consumes and `postgres` implements it. Business rules live in services and are unit-tested against in-memory fakes. The real SQL is exercised by the API suite.

## API surface

| Area | Endpoints |
|---|---|
| Health | `GET /health` |
| Auth | signup · verify email · resend verification · login · login by emailed code · refresh · logout · forgot and reset password · `GET /api/auth/me` |
| Profile | `PATCH /api/me/profile` · `GET /api/hikers/{id}` (public card, no email) |
| Outings | `GET /api/outings` (public, paginated) · `GET /api/outings/{id}` (public, viewer-aware) · create · `PATCH` · cancel |
| Membership | ask to join · edit own request · withdraw · accept · decline · remove member · restore · host's request lists · `GET /api/me/outings` |
| Comments | list · add · delete · like · unlike |
| Messaging | conversations · messages · mark read · start a DM · accept, decline and reopen a DM |
| Notifications | list · mark one read · mark all read |
| Live updates | `GET /api/events` (SSE) |

Design notes worth knowing:

- **Reads are public, writes are authed.** `GET /api/outings/{id}` works anonymously. A logged-in viewer also gets `my_request`, their own standing on the outing.
- **Capacity gates acceptance, not asking.** Anyone can ask. The host's accept fails with `409` if the person and their guests don't fit, checked atomically in SQL so concurrent accepts can't oversell.
- **Seat math is derived, never stored.** `people_count`, `seat_capacity`, `seats_short` and `spots_left` are computed per read. A host may shrink capacity below commitments; the shortage shows in the detail view and a human resolves it.
- **Keyset pagination.** The browse list returns `{items, next_cursor}`, 20 per page and 50 at most. The cursor is opaque and encodes the `(starts_at, id)` of the last row, so every page is one index scan.
- **Outing changes are pokes.** The `outing.changed` event carries only the outing's id. The page refetches through the normal endpoint, so permissions are checked in one place.
- **Cookie sessions.** An `HttpOnly` access token and an `HttpOnly` rotating refresh token, one session per login, so two browsers don't sign each other out. A refresh token one rotation old is accepted for 10 seconds to absorb races.
- **An expired session is not a stranger.** On the public outing route, a request with a refresh cookie and no valid access token gets `401`, and the client refreshes and retries. A refresh the server refuses clears both cookies.
- **Status codes follow one rule.** `400` the input is malformed, `403` you are not allowed, `409` the thing is in the wrong state.
- Responses use a uniform envelope: `{data, correlation_id, timestamp}` or `{error: {...}}`.

## Running locally

Requirements: Go 1.26+, PostgreSQL 16, `goose`, Node LTS.

```bash
# backend/.env
DATABASE_URL=postgres://...
JWT_ACCESS_SECRET=...        # openssl rand -base64 48
JWT_REFRESH_SECRET=...       # must differ from the access secret
APP_ENV=dev
APP_PORT=8088
```

```bash
cd backend
make migrate-up   # apply migrations to DATABASE_URL
make check        # gofmt + golangci-lint + build + unit tests (race detector on)
make run          # boots on :8088 (make dev runs it under air, with reload)
make e2e          # Playwright API suite; the server must be running
```

```bash
cd frontend
npm install
npm run dev       # Vite dev server on :5173
npm run e2e       # Playwright browser suite; API and dev server must be running
```

Optional backend env: `RATE_LIMIT_RPS` and `RATE_LIMIT_BURST` (defaults 10 and 20), JWT expiries, `DB_*` pool tuning, `APP_DISPATCHER_INTERVAL`, `CORS_ORIGINS`, and `RESEND_API_KEY` (required only in production). See `backend/internal/config`. The frontend reads `VITE_API_URL`.

## Testing

Three suites, each proving what the others can't:

- **Unit** (`go test ./... -race`): 236 tests over the service layer against in-memory fakes: the request state machine, the capacity table, visibility, validation, pagination, sessions. Rules are written test-first.
- **API** (`backend/e2e`): 116 Playwright tests speaking real HTTP to a real database. This is where the SQL and the JSON are proven. Each test creates its own actors with unique emails, so there is no shared state and no cleanup.
- **Browser** (`frontend/e2e`): 70 Playwright tests, each run on a desktop viewport and a phone-sized one.

**CI:** every pull request to `main` runs lint and unit tests, then starts a throwaway PostgreSQL container, migrates it, boots the server and runs both Playwright suites. Nothing in CI touches production. Green gates the merge.

**Deploy:** a push to `main` that changes `backend/` runs the migrations against production through an SSH tunnel, ships the new binary, restarts the service and checks `/health`. Migrations run before the new binary, so they are expand-only.

## Docs

- [`docs/ROADMAP.md`](docs/ROADMAP.md): what shipped, what is left, and the ledger of known gaps
- [`docs/ops/production-database.md`](docs/ops/production-database.md): where production's database lives, backups, restore
- [`backend/docs/`](backend/docs): one design note per feature, with numbered decisions, and the engineering notes
