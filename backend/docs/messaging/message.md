# Muster — Messaging
*As of 2026-10-03 (rev 6 — as built on `feat/dm`) · spec = what · see MUSTER-ROADMAP.md for why/when*

Scope: v1 roster-scoped outing chat and DMs, on one primitive. Rev 5 was the design before code; rev 6 records what was built and where it differs. Changes to this doc go through a PR like code.

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

**Status at rev 6.** D1–D12 are implemented. Two notes:

- **D12 as built:** `EventsProvider` handles `EventSource.onerror` by closing the stream and calling refresh. 200 → reopen; 401 → stop; anything else → retry with backoff (1 s doubling to 30 s).
- **D4 gap:** D4 says reconnect refetch covers dropped pokes. Reopening the stream does not invalidate any query today, so a poke sent while the stream is down is recovered only by the next poke or a window-focus refetch. See §5.

---

## 2. Schema

One primitive for outing chat and DMs. As migrated in `20260919050620_init_conversations_messages.sql`:

```
conversations
  id             UUID PK
  kind           TEXT NOT NULL CHECK (kind IN ('outing','dm'))
  outing_id      UUID NULL UNIQUE REFERENCES outings(id) ON DELETE CASCADE   -- kind='outing' only
  dm_a           UUID NULL REFERENCES hikers(id) ON DELETE CASCADE           -- kind='dm' only; dm_a < dm_b
  dm_b           UUID NULL REFERENCES hikers(id) ON DELETE CASCADE
  dm_initiator   UUID NULL REFERENCES hikers(id)
  dm_status      TEXT NULL CHECK (dm_status IN ('pending','accepted','declined'))
  dm_declined_by UUID NULL REFERENCES hikers(id)
  created_at     TIMESTAMPTZ NOT NULL
  CHECK ((dm_status = 'declined') = (dm_declined_by IS NOT NULL))            -- unnamed; see ledger
  CONSTRAINT conversations_kind_shape CHECK (
       (kind='outing' AND outing_id IS NOT NULL AND dm_a IS NULL AND dm_b IS NULL
                      AND dm_initiator IS NULL AND dm_status IS NULL)
    OR (kind='dm'     AND outing_id IS NULL AND dm_a IS NOT NULL AND dm_b IS NOT NULL AND dm_a < dm_b
                      AND dm_initiator IN (dm_a, dm_b) AND dm_status IS NOT NULL))
  UNIQUE (dm_a, dm_b)

messages
  id              UUID PK
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE
  hiker_id        UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE
  seq             BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE         -- D6
  body            TEXT NOT NULL
  created_at      TIMESTAMPTZ NOT NULL
  INDEX (conversation_id, seq)
```

**Deletes are hard deletes.** Rev 5 planned a `deleted_at` soft-delete; the table has no such column and the store runs `DELETE FROM messages`. Consequences are listed in §5.

**Outing membership is derived at query time**: host + accepted `join_requests`. Never copied into a participants table (that would be a second home for the roster). Membership branches on `kind`: `outing` → roster query; `dm` → `hiker IN (dm_a, dm_b)`.

**Outing conversation row is created on outing creation** (same transaction).

**DM state machine** (service: `StartDM`, `AcceptDM`, `DeclineDM`, `ReopenDM`):

| From | To | Who | Notes |
|---|---|---|---|
| (none) | `pending` | initiator, through a CanDM door (§4) | `dm_initiator` set |
| `pending` | `accepted` | recipient only | initiator → 403; not pending → 409 |
| `pending` | `declined` | either party | `dm_declined_by` = actor |
| `accepted` | `declined` | either party ("close") | `dm_declined_by` = actor |
| `declined` | `accepted` | only `dm_declined_by` ("reopen") | column cleared; anyone else → 403 |

No other transitions; nothing is deleted. A non-party gets 403 on every transition; a non-DM conversation gets 400.

**DM lookup: normalized pair.** The service sorts the two hiker IDs, then the store runs `INSERT … ON CONFLICT (dm_a, dm_b) DO NOTHING RETURNING …` and falls back to a `SELECT` when nothing is returned. `GetOrCreateDM` reports `created`. No check-then-act; the DB serializes the duplicate-DM race.

**Starting a DM that already exists returns the existing row, whatever its status**, with no poke and no notification. This differs from rev 5, which said a re-request after a decline returns 403. A declined DM stays declined until its closer reopens it; the declined row is still the block, and there is no block table.

---

## 3. Event contract (`GET /api/events`)

Auth: cookie (`withCredentials`). Membership is checked **per event at publish time** (D8), not at connect.

### Event types

| Event | Sent to | Data |
|---|---|---|
| `message.created` | every member of the conversation, author included | conversation poke |
| `message.deleted` | every member of the conversation | conversation poke |
| `dm.requested` | both parties | conversation poke |
| `dm.accepted` | both parties | conversation poke |
| `dm.declined` | both parties | conversation poke |
| `dm.reopened` | both parties | conversation poke |
| `notification.created` | the one recipient | notification poke |

```
event: message.created
data: {"conversation_id":"<uuid>","conversation_kind":"outing","outing_id":"<uuid>"}

event: dm.accepted
data: {"conversation_id":"<uuid>","conversation_kind":"dm"}

event: notification.created
data: {"kind":"join_request_created","outing_id":"<uuid>"}

event: notification.created
data: {"kind":"dm_requested","conversation_id":"<uuid>"}

: ping                                     -- heartbeat, D11
```

- **Conversation poke:** `conversation_id`, `conversation_kind` (`outing` | `dm`; renamed from `kind` in rev 6), and `outing_id` for outing conversations only (omitted for DMs).
- **Notification poke:** `kind` plus `outing_id` for outing kinds or `conversation_id` for DM kinds.
- The stream never carries message bodies or notification text (D3).
- The client registers listeners for exactly the seven types in `EVENT_TYPES` (`EventProvider.tsx`). A new event type must be added there or it is silently ignored.

### Client dispatch: query-key map

| Query key | Fetches | Invalidated by events | Invalidated by mutations |
|---|---|---|---|
| `['messages', cid]` | `GET /api/conversations/{cid}/messages` | `message.created`, `message.deleted` where `conversation_id === cid` | post message, delete message |
| `['conversation', cid]` | `GET /api/conversations/{cid}` | the four `dm.*` where `conversation_id === cid` | accept, decline, reopen |
| `['conversations']` | `GET /api/conversations` (inbox) | on the conversation page: the four `dm.*` for that `cid`. On `/inbox`: `message.created`, `message.deleted` and the four `dm.*`, unfiltered | start DM, accept, decline, reopen |
| `['notifications']` | `GET /api/notifications` (bell) | `notification.created` | mark read, read all |
| `['outing', id]`, `['outings']`, `['my-outings']`, `['outing-join-requests', id]` | outing detail page | `notification.created` where `outing_id === id` | (outing mutations) |

Subscriptions live in `useConversationEvents(cid)` (conversation page), `inbox.tsx`, `NotificationBell.tsx` and `outings.$id.tsx`.

### Delivery

Hub uses non-blocking send with drop (`select { case c.Send <- ev: default: }`). A dropped poke is recovered by the next poke or a window-focus refetch; see the D4 gap in §1 and §5.

### HTTP surface

```
GET    /api/events
GET    /api/conversations                    inbox; list is never null
GET    /api/conversations/{id}               ConversationView; non-member → 403
GET    /api/conversations/{id}/messages
POST   /api/conversations/{id}/messages
DELETE /api/messages/{id}
POST   /api/dms                              {hiker_id} → conversation (new or existing)
POST   /api/conversations/{id}/accept        204
POST   /api/conversations/{id}/decline       204
POST   /api/conversations/{id}/reopen        204
```

- **`ConversationView`** (viewer-independent): the conversation row, `participants: [{hiker_id, name}]`, and `outing_title`, `outing_starts_at`, `outing_host_id` (null for DMs).
- **`ConversationSummary`** (inbox row, viewer-dependent): `id`, `kind`, `title` (outing title, or the other party's name for a DM), `dm_status`, `dm_initiator`, `dm_declined_by`, `last_message_at` (null when empty), `last_preview` (first 80 chars, `''` when empty), `created_at`. Ordered by `COALESCE(last_message_at, created_at) DESC`, so an empty conversation sorts by when it was created.

### Notifications from DMs

| Kind | Recipient | When |
|---|---|---|
| `dm_requested` | the other hiker | a DM is created (not when an existing one is returned) |
| `dm_accepted` | the initiator | recipient accepts |
| `dm_reopened` | the other party | the closer reopens |

Decline and close are silent: no notification, only the `dm.declined` poke. Stored payload: `{conversation_id, from_name}`. `from_name` is the actor's name at insert time; if the name lookup fails the row is still inserted with an empty `from_name` and the bell falls back to "someone". There is no bell entry per message (D7).

---

## 4. Outing-scoping and anti-spam rules (service layer; each rule = one red test)

**Both kinds**
- Body: 1–500 characters after trim; empty or whitespace-only → 400; longer → 400.
- Rate limit: 10 messages per hiker per minute per conversation → 429. The count is of rows in the last minute.
- Read and post: members only. Anyone else → 403.

**Outing chat**
- Members: host + roster (accepted `join_requests`).
- Outing `open` → post allowed. Any other status → 403 (read stays until the v2 close-room decision).
- Delete: the author, or the outing's host.
- D5's N=100 is a pagination trigger, **not** a cap. No per-outing message ceiling.

**DMs**
- **CanDM doors** (checked when a DM is started; a self-DM → 400):
    1. Both hikers belong to the same outing, each as host or accepted member.
    2. The initiator hosts an outing on which the other hiker has a pending request. One-way: the requester cannot open a DM with the host.
- Posting by status: `accepted` → allowed. `pending` → the initiator may have one opening message; the recipient gets 403. `declined` → 403 for both.
- Delete: the author only. There is no host moderation in a DM.
- Roster removal does **not** close an existing accepted DM. The doors are checked only at start.
- Either party may close (`accepted → declined`); only the closer may reopen. No separate `closed` status.

---

## 5. Known limitations (recorded, not bugs)

1. **`seq` is insert-time, not commit-time.** Two concurrent inserts can commit in reverse `seq` order. Under D4 (load-all) both appear; nothing is lost. Only matters if a cursor is ever introduced.
2. **Dropped pokes** (hub buffer full, or the stream is down) delay visibility until the next poke or a window-focus refetch. **Reopening the stream does not refetch** (the D4 gap). On the ledger.
3. **A live stream outlives its token's `exp`.** By design: the server checks auth at connect only. Roster removal is handled by D8.
4. **DM threads unbounded** (D5).
5. **Single instance** (D9): a second API instance would split the hub. Must be revisited before horizontal scaling.
6. **Hard deletes reset counters.** The pending-DM "one opening message" rule and the rate limit both count existing rows. An initiator who deletes their opening message can send another while still pending. Each one reaches the recipient's inbox preview; none rings the bell.
7. **DM notifications are emailed with the generic template.** The email dispatcher drains every notification row; it has no content for the three DM kinds, so they fall through to "You have a new notification" and log `contentFor: unhandled kind`.
8. **Notification kinds have several homes**: see the checklist in §7.

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

**Added with DMs (rev 6)**

- Service tests cover the state machine in §2 and the posting rules in §4. Handler tests run against a fake service (`api.messageService`).
- Store SQL (CanDM, GetOrCreateDM, the inbox query, ConversationView) is proven by hand in psql and through the API suite, not by Go integration tests.
- API suite `dm.spec.ts`: 64 passing at rev 6.
- Playwright `message.spec.ts`, "host DMs a pending requester": two contexts, nothing reloads. Proves the bell poke, the inbox row arriving by `dm.requested`, `dm.accepted` reaching the initiator's open page, and messages crossing both ways. Two sabotages witnessed: skipping Accept, and removing the inbox's `dm.requested` subscription.

---

## 7. Checklist: adding a notification kind

A kind lives in seven places. Miss one and it fails quietly.

1. **Go const** in `internal/notification/notification.go`.
2. **`Kind.Valid()`** in the same file: add it to the switch.
3. **DB CHECK** `notification_events_kind_check`: a new migration that widens the list. The deploy runs migrations before the new binary starts, so the migration must be safe under the old binary (widening is).
4. **Stored payload**: decide the keys and write them in the service's `notify`. Outing kinds carry `{outing_id, outing_title}`; DM kinds carry `{conversation_id, from_name}`.
5. **Poke**: the `notification.created` data carries `kind` plus the id the client filters on.
6. **Email**: add a case to `contentFor` in `internal/notification/dispatcher.go`, or accept the generic fallback on purpose.
7. **Frontend bell** in `NotificationBell.tsx`: the text in `notificationText`, and the click target (conversation if `conversation_id`, else outing if `outing_id`).

Then: one service test that the right recipient gets the kind, and one line in the API suite.

For a new **event type** on the stream (not a notification kind): add it to `EVENT_TYPES` in `EventProvider.tsx`, subscribe where needed, and add a row to the tables in §3.

---

## Ledger additions from this doc
- Read tracking (badges, receipts, `last_read`, `last_fetched`) — cut for v1, don't re-raise.
- Pagination — v2, triggered by D5's N.
- Multi-instance fan-out — v2+.
- Refetch on stream reopen (the D4 gap).
- Name the unnamed declined-by CHECK on `conversations` (new migration; the table is in prod).
- Email content for DM kinds, or exclude them from the dispatcher.
- Pending-DM opening message can be repeated by delete-and-resend (§5.6).
- Bell entry per message and unread badges — v2, one feature with read tracking.
- Presence via hub count — v2.
- Outing notification kinds could carry the actor's name, as DM kinds do.
- Fake `ListConversations` in the service tests is a stub.