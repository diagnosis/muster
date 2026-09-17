# Muster — Messaging
*As of 2026-09-16 (rev 5 — final before code) · spec = what · decisions before code · see MUSTER-ROADMAP.md for why/when*

Scope: v1 roster-scoped outing chat, then DMs, on one primitive. No open items. Changes to this doc go through a PR like code.

---

## 1. Decisions (settled 2026-09-16)

| # | Decision | Why (Muster-specific) | Rejected |
|---|---|---|---|
| D1 | **Transport: SSE** | Server→client only (sends are `POST`). Delay-tolerant domain. Cookie auth rides `EventSource(withCredentials)`. Browser auto-reconnect is free. | WS: bidirectional buys nothing, needs library + framing + ping/pong + Nginx `Upgrade`. Pure polling: chat is v1's headline; hub is needed for DMs anyway. |
| D2 | **One stream per user**: `GET /api/events`, typed events | Browsers cap ~6 HTTP/1.1 sockets per host; one `EventSource` per feature would starve API calls. One handler, one Nginx `location`, one Flusher check. | One stream per feature. |
| D3 | **Poke-only events** — the stream never carries message bodies | One source of truth (DB via `GET`). No dedup, no `Last-Event-ID` bookkeeping. Refetch is idempotent, so a dropped poke is harmless *if* D4 holds. | Payload events: ordering + duplicate delivery become client problems. |
| D4 | **Client: poke → invalidate → refetch whole thread** (TanStack Query) | Load-all is exact; no "missed the middle" case. Reconnect/focus refetch covers dropped pokes. | Cursor/`?after=` (see D5). |
| D5 | **Load-all, no pagination in v1.** Assumed ceiling: **N = 100** messages per outing conversation | Roster 5–20 people over a few weeks. Pagination is the v2 trigger when N is exceeded. **DM threads have no end date — known risk, not solved in v1.** | Keyset pagination. |
| D6 | **Ordering column: `seq BIGINT GENERATED ALWAYS AS IDENTITY`** on `messages`, `ORDER BY seq` | PKs are UUIDv4 (random) — `ORDER BY id` is no order. `created_at` has ties and is tx-start, not commit time. `seq` has no ties and no clock. | `created_at`; UUID id. |
| D7 | **No read tracking of any kind in v1** — no `last_read`, no unread badge, no receipts | Product call: this isn't for close friends. A "server saw a GET" signal lies under D4 (background tabs refetch). Reopenable later as one column + one PATCH. **On the ledger: don't re-raise.** | `unread_count` column (two-homes), `last_read_message_id`, `last_fetched_at`. |
| D8 | **Hub routes by `hiker_id`; it holds no roster.** Registry is a `sync.RWMutex`-guarded `map[hikerID]map[*client]struct{}` (no actor loop, no register/unregister channels). Per-client buffered `Send` channel; publish is a non-blocking send under `RLock`. Only the stream goroutine writes to its `ResponseWriter`. The messaging *service* loads membership from the store at publish time and calls `BroadcastToUser` per member. | Hole B: membership checked at publish time, against the store. Host removes a hiker → next publish doesn't include them. No second home for the roster. | Hub keyed by outing (subscribe-time membership = stale). |
| D9 | **Single API instance for v1.** Hub is in-process. | DO droplet, one binary. Multi-instance fan-out (Postgres `LISTEN/NOTIFY`, Redis) is a v2+ decision, recorded here so it's not an omission. | — |
| D10 | **Nginx: dedicated `location /api/events`** modeled on the existing `/api/sse` block (`proxy_buffering off`, `proxy_http_version 1.1`, `Connection ''`, long `proxy_read_timeout`). Handler also sends `X-Accel-Buffering: no`. | Current `muster-api` block is a bare `location /` with defaults: buffering on, 60s read timeout. SSE dies. Header is belt-and-braces against a future config edit. | — |
| D11 | **Heartbeat**: handler writes an SSE comment line (`: ping`) every **15 s** (< mobile NAT idle ~30–60 s, < Nginx read timeout) | Idle streams are cut by proxies; browser can't tell dead-TCP from quiet. | — |
| D12 | **Client owns stream lifetime (option b).** On `EventSource.onerror`: call refresh; on success, reopen the stream; on failure, stop (user is logged out). Also reopen after any successful refresh triggered by `request()`. Server does nothing special at token `exp` beyond rejecting the next connect with 401. | Hole A. `EventSource` retries only dropped connections; a **401 fails permanently, no retry**. Heartbeats are server→client and cannot trigger refresh; with no other fetch in flight, `onerror` is the only signal the client gets. | (a) server-side close at `exp`. |

---

## 2. Schema

One primitive for outing chat and DMs.

```
conversations
  id          UUID PK
  kind        TEXT CHECK (kind IN ('outing','dm'))
  outing_id   UUID NULL REFERENCES outings(id) ON DELETE CASCADE   -- kind='outing' only; UNIQUE
  dm_a        UUID NULL REFERENCES hikers(id) ON DELETE CASCADE    -- kind='dm' only; dm_a < dm_b
  dm_b        UUID NULL REFERENCES hikers(id) ON DELETE CASCADE
  created_at  TIMESTAMPTZ
  CHECK ((kind='outing' AND outing_id IS NOT NULL AND dm_a IS NULL AND dm_b IS NULL)
      OR (kind='dm' AND outing_id IS NULL AND dm_a < dm_b))
  UNIQUE (dm_a, dm_b)

messages
  id              UUID PK
  conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE
  hiker_id        UUID REFERENCES hikers(id) ON DELETE CASCADE
  seq             BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE         -- D6
  body            TEXT NOT NULL
  created_at      TIMESTAMPTZ
  deleted_at      TIMESTAMPTZ NULL                                    -- soft-delete, as comments
  INDEX (conversation_id, seq)
```

**Outing membership is derived at query time**: host + accepted `join_requests`. Never copied into `conversation_participants` (that would be a second home for the roster). The membership function branches on `kind`: `outing` → roster query; `dm` → `hiker IN (dm_a, dm_b)`. One function, two branches, both tested.

**Outing conversation row is created on outing creation** (same transaction). The message page greets the roster on landing. No backfill migration: prod has no outings yet — **receipt required before any wipe: `SELECT count(*) FROM outings` on prod, recorded in the PR.**

**DM state**: `dm_status TEXT NULL CHECK (dm_status IN ('pending','accepted','declined'))` and `dm_initiator UUID NULL` on `conversations` (`dm` kind only; NULL for outing). A DM is created `pending` by the initiator; the other hiker accepts or declines; Add `dm_declined_by UUID NULL` (set whenever status becomes `declined`, cleared on `accepted`). Transitions: `pending → accepted` (recipient); `pending → declined` (recipient); `accepted → declined` (either party — this is "close"); `declined → accepted` (**only** `dm_declined_by`: unblock / reopen is a late accept by whoever closed). No other transitions; nothing is deleted. Test: initiator closes, recipient tries to reopen → 403; initiator reopens → accepted. Messaging: allowed when `accepted`; while `pending`, the initiator may send **exactly one** opening message, the recipient none. `declined` is terminal for the initiator: a re-request hits `UNIQUE (dm_a, dm_b)` → 403. The declined row *is* the block; no block table.

**DM lookup — option (b), normalized pair on `conversations`**: `dm_a UUID NULL, dm_b UUID NULL`, `CHECK (kind <> 'dm' OR dm_a < dm_b)`, `UNIQUE (dm_a, dm_b)`. Service sorts the two hiker IDs before insert. Create = `INSERT … ON CONFLICT (dm_a, dm_b) DO NOTHING RETURNING id`, then `SELECT` if nothing returned. No check-then-act; the DB serializes the duplicate-DM race. `conversation_participants` is **not needed in v1** (outing membership derived; DM membership is the pair). Tests: same pair passed in both orders resolves to one conversation (operand-swap class); two concurrent creates → `count(*)=1`, red before the constraint exists.

DM UI and rules are deferred; only the primitive is settled here.

Soft-delete and host moderation: **same rules as comments.** v2: on outing complete/cancel, close the room (delete all vs freeze read-only — decide then, not now).

---

## 3. Event contract (`GET /api/events`)

Auth: cookie (`withCredentials`). Roster/participant check is **per event at publish time** (D8), not at connect.

```
event: message.created
data: {"conversation_id":"<uuid>","kind":"outing","outing_id":"<uuid>"}

: ping                                     -- heartbeat, D11
```

Client dispatch: on `message.created`, `invalidateQueries(['conversation', conversation_id, 'messages'])`. Nothing else. Unknown event types are ignored.

Later event types ride the same stream (`notification.created` for the bell is the obvious next one — a separate decision, not v1 chat).

Delivery: hub uses non-blocking send with drop (`select { case c.Send <- ev: default: }`). A dropped poke costs nothing beyond D4's refetch-on-focus/reconnect.

---

## 4. Outing-scoping and anti-spam rules (service layer; each rule = one red test)

**Outing chat**
- Post/read: host + roster (accepted `join_requests`). Anyone else → 403.
- Outing `open` → post allowed. `cancelled` → 403 (read stays until the v2 close-room decision).
- Body: 1–500 chars after trim; empty/whitespace-only → 400.
- Rate limit: 10 messages per hiker per minute per conversation → 429. Sliding window or fixed bucket — store mechanics, service policy; fake mirrors the mechanics.
- D5's N=100 is a pagination trigger, **not** a cap. No per-outing message ceiling.

**DMs**
- Host and roster members of a shared outing may open a DM with each other; host may open one with a *pending* requester.
- State machine in §2 governs who may message when.
- Roster removal does **not** close an existing accepted DM.
- Either party may **close** a DM = `accepted → declined` with `dm_declined_by` = closer. Only the closer may reopen. No separate `closed` status.
- Same body cap and rate limit as outing chat.

## 5. Known limitations (recorded, not bugs)

1. **`seq` is insert-time, not commit-time.** Two concurrent inserts can commit in reverse `seq` order. Under D4 (load-all) both appear; nothing is lost. Only matters if a cursor is ever introduced.
2. **Dropped pokes** (hub buffer full) delay visibility until the next poke, focus, or reconnect. Acceptable under D3/D4.
3. **Stream outlives credential** until D12 is implemented. Roster *removal* is already handled by D8.
4. **DM threads unbounded** (D5).
5. **Single instance** (D9): a second API instance would split the hub. Must be revisited before horizontal scaling.

---

## 6. Test plan (red first, all three layers)

Hub (`internal/events`, mutex registry + per-client channel, ported ideas from DeployWatch minus the actor loop):
- register → broadcastToUser → client receives; other client does not
- unregister closes `Send`; double unregister does not panic
- full `Send` buffer → event dropped, hub does not block (assert with a timeout)
- **shutdown closes every client `Send`, stream handlers exit** (no goroutine leak; assert with `goleak` or a WaitGroup + timeout)
- heartbeat: handler writes `: ping` within interval with no events

Handler (`GET /api/events`):
- unauthenticated → 401, no stream
- headers: `text/event-stream`, `no-cache`, `X-Accel-Buffering: no`
- **`http.Flusher` reachable through every middleware in the chain** — assert, don't assume; use `http.NewResponseController(w).Flush()` (Go ≥1.20) if wrappers lack `Unwrap()`
- client disconnect (ctx done) → handler returns, client unregistered

Service (`POST .../messages`):
- non-member post → 403 (rule from §4)
- member post → row inserted, one `message.created` per roster member *including host*, none to non-members
- removed member (after `member_removed`) → receives nothing on next publish (Hole B)
- each §4 rule → one red test

Store:
- messages returned `ORDER BY seq`; a sabotage that orders by `created_at` must be caught by a test with a forced tie
- fake store mirrors the real store's mechanics (seq assignment, soft-delete filter), never policy

E2E (Playwright):
- two browser contexts, same outing: A posts, B's thread updates without reload
- B removed by host: A posts, B's thread does not update

Sabotage before green: mutate the roster check to always-true, watch the non-member test go red, restore.

---

## Ledger additions from this doc
- Read tracking (badges, receipts, `last_read`, `last_fetched`) — cut for v1, don't re-raise.
- Pagination — v2, triggered by D5's N.
- Multi-instance fan-out — v2+.