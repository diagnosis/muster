# Muster — Notifications Design (v1)

Status: accepted · Date: 2026-09-06

## Goal
Tell people what happened to their outings and requests — by email (when
they're away) and an in-app bell (when they're here). Nothing real-time,
nothing configurable, v1.

## Events

Derived from the join-request state machine and outing lifecycle — every
transition that affects someone who didn't cause it:

| event                   | recipient      | email | bell |
|-------------------------|----------------|:-----:|:----:|
| join_request_created    | host           |  ✓    |  ✓   |
| join_request_approved   | requester      |  ✓    |  ✓   |
| join_request_declined   | requester      |  ✓    |  ✓   |
| join_request_withdrawn  | host           |  ✓    |  ✓   |
| member_removed          | removed hiker  |  ✓    |  ✓   |
| outing_cancelled        | each member    |  ✓    |  ✓   |
| outing_updated          | each member    |  ✓    |  ✓   |

Event names byte-mirror the join-request status enum (two-homes rule).

**Parked (v2):** `new_outing_created` broadcast — unbounded fan-out; waits
for groups/subscriptions so "all" becomes a scoped audience. No v1 emails.

## Storage

One table; a row is both the bell item and the email queue entry:

    notification_events
      id          uuid PK default gen_random_uuid()
      hiker_id    uuid NOT NULL REFERENCES hikers(id) ON DELETE CASCADE  -- recipient
      kind        text NOT NULL CHECK (kind IN (...the seven...))
      payload     jsonb NOT NULL         -- outing_id, outing_title, actor_name, ...
      created_at  timestamptz NOT NULL default now()
      read_at     timestamptz NULL       -- bell: null = unread
      emailed_at  timestamptz NULL       -- outbox: null = email not yet sent

Multi-recipient events (cancelled/updated) insert one row per member —
fan-out at write time keeps reads and the dispatcher trivial.

## Delivery (outbox)

Services **insert only** — no email inside the request. A dispatcher
goroutine (started in main, ~30s ticker) drains:

    SELECT ... WHERE emailed_at IS NULL ORDER BY created_at LIMIT 50

renders via internal/email, sends via the mailer, stamps emailed_at on
success. Failures stay NULL → retried next tick. Send failures never
affect the triggering request (it already returned 200).

Insert happens in the same transaction as the state change where one
exists — a decline that commits always leaves its notification row.

## Bell API

    GET  /api/notifications           auth'd; recipient's rows, newest first, paged
    POST /api/notifications/{id}/read
    POST /api/notifications/read-all

Unread count rides the list response (or a cheap HEAD-style endpoint —
decide at implementation). Frontend: header bell + dropdown, badge from
unread count, invalidate on mark-read.

## Out of scope (v1)
Web push / service workers · live updates (SSE/websockets) · per-user
preferences · digests/batching · new-outing broadcast (v2, with groups)

## Slices
1. Migration + notification_events store + insert on decline (smallest event)
2. Dispatcher loop + email templates per kind (Content structs)
3. Remaining event inserts at their transition sites
4. Bell API + frontend dropdown