# `outing.changed` — outing pages and lists stay live

*Design for the first v1 item after PR #52. Lives at `backend/docs/events/outing-changed.md`. Decisions are numbered so tests and code can cite them.*

## Problem

Pokes ride on notifications. A waiter whose place changes because someone ahead withdrew gets no notification (nothing happened *to them*), so their page is stale until the tab regains focus. The host watching a roster, a member watching capacity, a signed-in hiker scanning the browse list for room — same thing. The client needs one signal: *this outing changed, refetch*.

## Decisions

| # | Decision | Why | Rejected |
|---|---|---|---|
| E1 | **One event type, `outing.changed`, payload `{"outing_id": "<uuid>"}`.** Poke only, no diff, no fields. | D3 from messaging: the DB is the one source of truth; the client refetches. Idempotent, so a dropped or duplicate poke is harmless. | A typed payload per transition (`member.joined`, `request.declined`…): nine event types the client must switch on, and a client that reconstructs state from events instead of reading it. |
| E2 | **Delivered to every open connection.** The hub gains `Broadcast(e Event)`, which walks all registered clients. The service calls it once per transition; it computes no audience. | The browse list is for hikers who are *not yet* on an outing, so audience fan-out would never reach the people the card capacity was built for. Everyone signed in gets an id; the detail is public anyway (capacity, counts). At Muster's scale — tens of connections, a few transitions an hour — the cost is nothing. | **Fan-out to the audience** (host ∪ roster ∪ pending): reuses the per-hiker hub but misses strangers on the page and everyone on the browse list, and needs a "who's the subject" rule for the declined hiker whose page must still flip. **Outing rooms in the hub**: a subscribe/unsubscribe protocol, cleanup on navigation, a second registry. Rooms are the right move when connection counts make `Broadcast` expensive; fan-out never is. |
| E3 | **Emit after the store call succeeds, never before; best-effort, never fails the request.** Same contract as `notify`: log and continue. | The client refetches on the poke; an emit before commit would refetch stale data. A broken hub must not turn a successful withdraw into a 500. | Emit inside the store transaction: couples the store to the hub. |
| E4 | **Every roster or request transition emits, plus `Update` and `Cancel`.** The list is the table below, and the Go test is a table over the same list. | `Update` changes `host_seats`/`max_size`/time, which changes capacity and the card. `Cancel` changes everything. The transition you forget is the one that stays silent for a year; the test row is what makes forgetting loud. | Emitting only on roster changes: a host raising `max_size` would leave every waiter's "full" stale. |
| E5 | **The outing page subscribes to `outing.changed` only, and drops its `notification.created` subscription.** `notification.created` stays the bell's and inbox's concern. | One signal per concern. E2 guarantees every open page gets `outing.changed`, so the page no longer needs to eavesdrop on notifications. | Keep both: two pokes for the same refetch, and "which one didn't fire?" on every future bug. |
| E6 | **The outing page filters by `outing_id`; the browse and My-outings pages don't filter.** Outing page invalidates `['outing', id]` and `['outing-join-requests', id]`; browse invalidates `['outings']`; My-outings invalidates `['my-outings']`. | Every tab gets every poke (E2); filtering is the client's job and costs one string compare. A list can't know whether the changed outing is on it without the data it's about to refetch, so it just refetches. | Server-side filtering by what page is open: that's rooms (E2). |
| E7 | **Anonymous viewers get nothing.** The stream is authenticated; a logged-out browse page refetches on focus as today. | The stream's auth model is not this feature's to change. | A public stream: a separate design, if ever. |
| E8 | **Comments don't emit `outing.changed`.** | They have no effect on roster or capacity and already have their own refetch path. Scope. | — |

## Transitions (E4) — this table *is* the test table

| Transition | Actor | What a page shows differently |
|---|---|---|
| `Update` | host | capacity line, bar, date, meeting point |
| `Cancel` | host | cancelled chip, actions gone |
| `RequestJoin` | hiker | waiting count, waiting list, host's requests list |
| `EditJoinRequest` | hiker | capacity (seats/guests), waiting list summary |
| `Withdraw` | hiker | roster or waiting list, capacity |
| `Accept` | host | roster, waiting list, capacity |
| `Decline` | host | waiting list; the declined hiker's page loses its place |
| `RemoveMember` | host | roster, capacity; the removed hiker's page |
| `RestoreRequest` | host | waiting list; the restored hiker's page regains its place |

Nine rows. Each row's test: seed what the transition needs, run it, assert the fake broadcaster recorded exactly one `outing.changed` whose data carries this `outing_id`. A tenth row, `AddComment`, asserts zero (E8).

Sabotage: delete the emit from one transition → exactly that row red. Move an emit above its store call and make the store fail → that row shows an emit with no change behind it (E3).

## Shape (not code)

**Hub.** `Broadcast(e Event)` walks every client in every set with the same non-blocking send as `BroadcastToUser`. `Broadcaster` interface gains the method; the fake gains it and a `broadcast() []Event` accessor beside `sentTo`.

**Service.** `broadcastOutingChanged(ctx, outingID)` marshals `{"outing_id"}` and calls `Broadcast`. Nine call sites, one line each, after the store call, beside the existing `notify` where there is one.

**Client.** `"outing.changed"` appended to `EVENT_TYPES`. `outings.$id.tsx`: the `useEffect` at line 37 changes its event name and keeps the id filter and the two detail-page keys; the list keys move out of it. Browse and My-outings routes each gain a three-line `useEffect` that subscribes and invalidates their own key.

**Known double-poke.** For the person a notification is *about*, `notify` still sends `notification.created` (bell) and the transition sends `outing.changed` (page). Two listeners, one refetch each; React Query dedupes an in-flight fetch for the same key. Accepted.

## Tests

| Tier | What | Red first? |
|---|---|---|
| Full | `Test_Hub_Broadcast` in `events`: two hikers, three clients (one hiker has two tabs), one unregistered → three receive, one doesn't; a client with a full buffer is skipped, not blocked | yes — compile error against a method that doesn't exist |
| Full | `Test_OutingChanged_Table`: the ten rows above | yes — the fake records nothing until the emit exists |
| Light | Browser: two contexts on the same outing, waiter B withdraws, host A's capacity line changes with no reload and no focus event. Assert with `expect(...).toHaveText`, never a sleep. | optional; recommended sabotage is commenting out the client subscribe |
| Light | Browser: signed-in C on the browse list, B withdraws elsewhere, C's card line changes | optional |
| None | `EVENT_TYPES` string, doc comments | — |

No API-suite spec: an SSE assertion over Playwright's request API isn't worth the fixture, and `event_test.go` already covers the stream itself.

## Pre-existing bug, not this feature

`isAudience` includes the host; `notifyOutingAudience` doesn't. They are two homes for "who belongs to this outing" and disagree. Fix is one `audience(ctx, outing)` helper used by both, with a test (host + 2 accepted + 1 pending + 1 declined → 4 ids). Ledgered; one commit on any branch.

## Out of scope, ledgered

- Login cross-tab broadcast (separate item, same hub).
- Rooms, when `Broadcast` shows in a profile. Not before.
- A public stream for anonymous viewers (E7).