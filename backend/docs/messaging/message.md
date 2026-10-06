# Muster — Messaging
*As of 2026-10-06 (rev 8 — as built, including unread counts) · spec = what · see MUSTER-ROADMAP.md for why/when*

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
| D7 | **Read tracking is a per-member read marker; unread counts are derived from it** (rev 7, replaces "no read tracking in v1"). No read receipts: nobody sees whether someone else has read. Design in §8. | Without any signal a new message is invisible unless the inbox happens to be open. The bell stays for events, not messages; email per message is too much. Rev 5's two warnings still hold and shape §8: a "server saw a GET" signal lies under D4 (background tabs refetch), and a stored count is a second home. | Stored `unread_count` column (two-homes); marking read on `GET`; bell entry or email per message; read receipts. |
| D8 | **Hub routes by `hiker_id`; it holds no roster.** Registry is a `sync.RWMutex`-guarded `map[hikerID]map[*client]struct{}` (no actor loop, no register/unregister channels). Per-client buffered `Send` channel; publish is a non-blocking send under `RLock`. Only the stream goroutine writes to its `ResponseWriter`. The messaging *service* loads membership from the store at publish time and calls `BroadcastToUser` per member. | Hole B: membership checked at publish time, against the store. Host removes a hiker → next publish doesn't include them. No second home for the roster. | Hub keyed by outing (subscribe-time membership = stale). |
| D9 | **Single API instance for v1.** Hub is in-process. | DO droplet, one binary. Multi-instance fan-out (Postgres `LISTEN/NOTIFY`, Redis) is a v2+ decision, recorded here so it's not an omission. | — |
| D10 | **Nginx: dedicated `location /api/events`** modeled on the existing `/api/sse` block (`proxy_buffering off`, `proxy_http_version 1.1`, `Connection ''`, long `proxy_read_timeout`). Handler also sends `X-Accel-Buffering: no`. | Current `muster-api` block is a bare `location /` with defaults: buffering on, 60s read timeout. SSE dies. Header is belt-and-braces against a future config edit. | — |
| D11 | **Heartbeat**: handler writes an SSE comment line (`: ping`) every **15 s** (< mobile NAT idle ~30–60 s, < Nginx read timeout) | Idle streams are cut by proxies; browser can't tell dead-TCP from quiet. | — |
| D12 | **Client owns stream lifetime (option b).** On `EventSource.onerror`: call refresh; on success, reopen the stream; on failure, stop (user is logged out). Also reopen after any successful refresh triggered by `request()`. Server does nothing special at token `exp` beyond rejecting the next connect with 401. | Hole A. `EventSource` retries only dropped connections; a **401 fails permanently, no retry**. Heartbeats are server→client and cannot trigger refresh; with no other fetch in flight, `onerror` is the only signal the client gets. | (a) server-side close at `exp`. |

**Status at rev 8.** D1–D12 are implemented. D7 was reopened on 2026-10-04 and built as §8. Two notes on what is built:

- **D12 as built:** `EventsProvider` handles `EventSource.onerror` by closing the stream and calling refresh. 200 → reopen; 401 → stop; anything else → retry with backoff (1 s doubling to 30 s).
- **D4 as built:** every time the stream opens, the first open included, `EventsProvider` refetches every query on screen through `invalidateFresh` (§8, "The first-load race"). A poke sent before the stream connected, or while it was down, is made up by that refetch. Rev 6 and rev 7 skipped the first open; that left a gap between a page's first fetch and its stream connecting. Covered by the Playwright spec "validate inbox counts".

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

Hub uses non-blocking send with drop (`select { case c.Send <- ev: default: }`). A poke dropped because the hub buffer was full is recovered by the next poke or a window-focus refetch. Pokes missed while the stream was down are recovered by the refetch on reopen (D4 as built, §1).

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

Decline and close are silent: no notification, only the `dm.declined` poke. Each of the three kinds also sends an email with its own subject and a link to `/conversations/{id}` (`contentFor` in the dispatcher). Stored payload: `{conversation_id, from_name}`. `from_name` is the actor's name at insert time; if the name lookup fails the row is still inserted with an empty `from_name` and the bell falls back to "someone". There is no bell entry per message (D7).

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
- Delete: the author only, and only once the DM is no longer `pending`. While pending, nothing can be deleted (403), so the initiator's one opening message stays spent. There is no host moderation in a DM.
- Roster removal does **not** close an existing accepted DM. The doors are checked only at start.
- Either party may close (`accepted → declined`); only the closer may reopen. No separate `closed` status.

---

## 5. Known limitations (recorded, not bugs)

1. **`seq` is insert-time, not commit-time.** Two concurrent inserts can commit in reverse `seq` order. Under D4 (load-all) both appear; nothing is lost. Only matters if a cursor is ever introduced.
2. **Dropped pokes** (hub buffer full) delay visibility until the next poke or a window-focus refetch. Pokes missed before the stream connected or while it was down are made up by the refetch on open (D4 as built, §8).
3. **A live stream outlives its token's `exp`.** By design: the server checks auth at connect only. Roster removal is handled by D8.
4. **DM threads unbounded** (D5).
5. **Single instance** (D9): a second API instance would split the hub. Must be revisited before horizontal scaling.
6. **Hard deletes reset the rate limit.** The limit counts rows in the last minute, so deleting a message frees a slot. The pending-DM "one opening message" rule also counts rows, which is why deleting is refused while a DM is pending (§4).
7. **Every notification kind is emailed.** The dispatcher drains every notification row. A kind with no case in `contentFor` falls through to a generic "You have a new notification" email and logs `contentFor: unhandled kind`.
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

- Service tests cover the state machine in §2 and the posting and delete rules in §4. A table test covers the email content for the three DM kinds. Handler tests run against a fake service (`api.messageService`).
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

## 8. Unread counts (built in rev 8)

**What the user gets**

- The header's Inbox link shows the total number of unread messages across all of the viewer's conversations.
- Each inbox row shows that conversation's unread count, for DMs and outing chats alike.
- DM requests do not add to the count as requests; they already ring the bell. A request's opening message is a message and counts as one.
- Nothing changes in the bell, and no email is sent per message.
- On a phone the count sits on an inbox icon in the top bar, next to the bell. On desktop it sits on the Inbox link. The phone menu has no Inbox row.

**The marker**

```
conversation_reads
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE
  hiker_id        UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE
  last_read_seq   BIGINT NOT NULL
  updated_at      TIMESTAMPTZ NOT NULL
  PRIMARY KEY (conversation_id, hiker_id)
```

One rule everywhere: **unread = messages in the conversation with `seq` greater than my marker, that I did not write.** No marker means zero, that is, from the beginning. The count is computed in the inbox query; nothing stores it. A deleted message drops out of the count by itself.

**Where a marker comes from**

| Moment | Marker |
|---|---|
| A hiker's join request is accepted | set to the conversation's current highest `seq` (0 if it has no messages), in the same statement as the accept. Earlier messages stay readable and are not counted. |
| Accepted again after withdrawing | the same write moves the marker up to the current highest `seq`. |
| Host at outing creation; both people in a new DM | none. No marker means from the beginning, which is correct for them. |
| Deploy day | the migration backfills a marker at each conversation's current highest `seq` for every existing host, accepted member and DM pair, so history does not show up as unread. |
| The member reads | see "Marking read" below. |

The accept is one SQL statement in the outing store (`AcceptIfCapacity`), so capacity cannot be oversold. The marker is written inside that same statement: the `UPDATE` is wrapped as `WITH accepted AS (… RETURNING outing_id, hiker_id, updated_at)`, a second block inserts the marker from `accepted`, and the statement still returns `updated_at`. If the update accepts nothing, no marker is written. The Go code around it did not change.

**Marking read**

`POST /api/conversations/{id}/read` with `{"seq": <n>}` → 204.

- A negative `seq`, a bad conversation id or broken JSON → 400, and the service is never called.
- Members only; anyone else → 403 and nothing is written.
- The service clamps `seq` to the conversation's highest `seq` (`min(maxSeq, seq)`), so a marker cannot point past what exists.
- The store's upsert only moves the marker forward: the stored value becomes `GREATEST(stored, incoming)`. A late or repeated call cannot move it back. In short: the service handles beyond, the store handles below.
- No event is broadcast. Reading is private.

**When the client calls it** (D7's rule: reading is an explicit signal, never a side effect of a `GET`)

- `useMarkReadWhenSeen(cid, lastSeq)`, called from `Chat`. When the conversation is open and the tab is visible, it reports the highest `seq` it is showing: on load, and again whenever a new message arrives.
- A tab in the background does not report. When it becomes visible again, it reports then.
- On success the client refetches `['conversations']` through `invalidateFresh`, so the row count and the header total drop.

**Reading the counts**

- `ConversationSummary` gains `unread_count`. The header total is the sum over the same `['conversations']` data; there is no second endpoint.
- `InboxEvents`, mounted once in the logged-in header, subscribes to the six events that can change the inbox (`message.created`, `message.deleted` and the four `dm.*`) and refetches `['conversations']`, so the total moves on any page. The inbox page has no subscriptions of its own.
- No new event types.

**The first-load race, and `invalidateFresh`**

Invalidating a query while it is on its very first fetch does not start a new request: TanStack Query reuses the request in flight, and that request may have been answered before the change. On a freshly loaded page this left the inbox count stale about 4 times in 100 runs, and it was the cause of earlier flakes in the delete-message and join-request specs.

`invalidateFresh(qc, filters)` in `src/queries.ts` checks whether a matching query is on its first fetch, invalidates, and if so invalidates once more after that fetch has finished. It is used by the stream's `onopen`, by `InboxEvents`, and by mark-read's `onSuccess`. With it, "validate inbox counts" passed 100 of 100 runs.

**Rules tested**

Service (`MarkRead`, against the fake store):
- marking read as a non-member → 403, no marker written
- a lower `seq` after a higher one leaves the marker unchanged
- a `seq` beyond the conversation's highest is clamped
- a normal mark stores the `seq` that was sent

Store (psql): the upsert with 479, then 600, then 100 left the marker at 600; `MaxSeq` matched the last message and returned 0 for an empty conversation; the backfill inserted 2,843 rows against 2,843 expected locally; the count expression gave 0 at the marker and 1 after lowering it.

Handler (`Test_HandleMarkRead`, fake service): 204 with the right arguments passed on; 400 for a negative `seq`, a bad id and broken JSON, with the service never reached; a service 403 passed through.

API suite (`message.spec.ts`): member marks read → 204; stranger → 403; "unread counts" (3 unread, own message does not count, mark read → 0, the other member sees 1); "unread count on accept" (history not counted, the next message is).

Playwright (`message.spec.ts`, "validate inbox counts"): hiker parked off the inbox; host posts; the header's Inbox count shows 1 with no reload; hiker opens the conversation; the count clears. Runs on both projects with no waits.

Not covered by a test: a background tab not marking read. Check it by hand.

**Known limits of this design**

- Every message poke makes each online member refetch the inbox list. Fine at D9's scale; revisit with multi-instance fan-out.
- A second tab of the same user keeps its old count until its next poke or focus.
- `seq` is insert-time (§5.1): a message that commits late with a `seq` below a marker set in between is never counted as unread. Rare, and the message is still shown.
- Someone who is not on the site learns nothing until they return. A once-a-day digest email is on the ledger as a maybe.

---

## Ledger additions from this doc
- Read receipts (who has read what) — cut, don't re-raise. Unread counts are in: §8.
- Once-a-day digest email of unread messages for people who are away — maybe, undecided.
- Pagination — v2, triggered by D5's N.
- Multi-instance fan-out — v2+.
- Move the remaining event-driven invalidations (conversation page, bell, outing page) to `invalidateFresh`.
- Accept after a withdrawal moving the marker forward has no test of its own.
- A cancelled request (client gone) is logged as a warning with a 500; log it quietly instead.
- Name the unnamed declined-by CHECK on `conversations` (new migration; the table is in prod).
- The chat still shows a delete button on a pending DM's message; the server refuses it.
- Bell entry per message — not planned; unread counts (§8) cover it.
- Presence via hub count — v2.
- Outing notification kinds could carry the actor's name, as DM kinds do.
- Fake `ListConversations` in the service tests is a stub.