# Muster — Roadmap

*As of 2026-10-14 · repo: github.com/diagnosis/muster · written from the code on `main` (PR #45 merged) and confirmed by Safa. Replaces the 2026-08-07 roadmap.*

*Doc hierarchy: roadmap = why and when · spec = what · engineering notes = how.*

---

## Where things stand

Muster is live at muster.safadev.app. Backend and frontend are on `main`, gated by CI (`ci.yml`) and deployed by `deploy.yml`. Migrations run before the new binary, so schema changes are expand-only.

Test suites: Go unit tests against a fake store, a Playwright API suite (11 spec files) that proves the real SQL and JSON, and a Playwright browser suite (22 spec files).

### Shipped

| Area | What exists |
|---|---|
| Accounts | signup, email verification, login, logout, refresh with shared in-flight request, forgot and reset password, profile edit |
| Outings | create, edit, cancel, browse, detail, optional end time (max 14 days), My outings |
| Joining | request with role, seats and guests; withdraw; re-request applies the new details and goes to the back of the queue; host accept, decline, remove; capacity checked atomically at accept |
| Outing phases (PR #42) | upcoming, in progress, past, computed on the server; My outings tabs; roster and waiting list frozen from the start; host edits until 12 hours before the start; comments open until 7 days after the end |
| Discussion | comments with one level of replies, likes, soft delete |
| Messaging | outing group chat, direct messages with request and accept, inbox, unread counts, delete, close and reopen |
| Notifications | in-app bell, live updates over server-sent events, email for selected events |
| Waiting and withdrew lists (PR #43) | on the outing page for logged-in viewers, names only; waiting in the order people asked (`requested_at`), withdrew most recent first; the going list is sent by viewer level: count only when logged out, names when logged in, hiker ids for members, request ids for the host |
| Chat closure (PR #44) | outing chat is read-only 7 days after the outing ends, the same window as comments; history stays readable and delete stays allowed; the chat page shows the server's reason when a send is refused |
| Request history and restore (PR #45) | the host's requests list takes `?status=` (requested, declined, removed; anything else is 400); declined and removed requests can be restored to the queue at their original `requested_at`, host-only and only before the outing starts; the hiker gets a `join_request_restored` notification; the outing page shows Waiting, Declined and Removed tabs with a Restore button on history rows |

---

## v1 — what is left

In this order. Items can be added.

1. **Change password while logged in.** Verify the current password, rehash, revoke refresh tokens. Today only the emailed reset exists.
2. **Edit Pending Request** (role, seats, guests) without losing your place.
3. **Passwordless login with an emailed code.**
4. **Public hiker card page.** The API route exists (`GET /api/hikers/{id}`); the page does not. Home area and bio are in the backend and get surfaced here and on the profile form, together with a profile photo or avatar.
5. **Pagination.** Browse first, with keyset pagination. Then chat, then the Past tab. Server-side phase filtering comes with it.
6. **Audit.** After the items above: every list endpoint checked for limits, every permission rule checked against its test, error codes made consistent.
7. **One real hike organized through Muster.** v1 is done when boots hit dirt.

Also open from the original v1 list: `capacity_state` served by the API, so the client stops deriving it.

### Polish ledger (not blocking v1; do in batches)

- Forms: login, signup, profile, create and edit outing.
- Outings list and card: one shared `OutingMeta` component for the card and the detail page.
- Detail page labels on a started outing ("Requested — waiting on host", "You're going!").
- Edit button still shows inside the 12-hour cutoff; the server refuses with a clear message.
- `Cancel` and `Update` return 400 for a started outing where other actions return 409.
- The guest limit is written into an error message as "0-3" and also lives in `maxGuests`.
- API specs that accept two or more riders should state their own capacity.
- Remaining cache invalidations moved to `invalidateFresh`.
- Requests cancelled by the client logged quietly, not as a 500 warning.
- Notes cannot be cleared on update.
- The conversation header shows only the start time.
- `window.confirm` for cancelling an outing replaced with the modal.
- Unnamed CHECK constraint on conversations given a name.
- Unused `Phase.Valid()` removed.
- Weekly digest email: maybe.
- Guest counts on the going and waiting lists ("Selin + 2").
- An injected clock in the outing service, as the message service has.
- Rename the test helper seedMember to seedHiker.
- An API spec for the withdrawn order with two people.
- ESLint errors that are also on main (3 errors, 5 warnings; CI doesn’t run the linter).
- Refetch the conversation after a refused send, so an open chat page catches up without a reload.
- Chat page: hide the input on a cancelled outing (the conversation view needs the outing’s status).
- backend/e2e/types.ts: ConversationResponse has dma_a where the server sends dm_a
- An API spec for the 7-day comment window, using setOutingTimes (it is proven in Go only)
- Undo removal in one click: restore, then try to accept, so the host is not sent back to the Waiting tab.
- Notification-kind migrations: the Down of the DM-kinds migration fails if rows of a removed kind exist; the restored-kind migration deletes them first. Backfill the older one the same way.
- The host card fetches all three request lists on load for the tab counts; one endpoint returning counts would do it in one request.
- The pending-request row has no visible cue that it opens a modal; a chevron or "Review" label would help.
- Full UI audit before v1 ships: one filled button per card, hover states on every custom control, button classes reconciled (`btn`, `btn-primary`, module overrides).

---

## v2 — the map, on the web

- Trail and meeting-point map on the outing page and on browse.
- No-shows: after a hike the host marks who did not come; counts appear on public stats, for riders as well as drivers. The frozen roster from v1 is what this is recorded against.
- Add-to-home-screen remains the phone story.

## v3 — mobile and trail-live

- Native app. Tracking is dropped from the web app and lives here.
- Offline-first group positions on pre-downloaded maps, shown honestly stale ("last seen 23 min ago").
- Push notifications.

The idea is unchanged: other apps sell solo navigation; Muster sells coordination, which falls out of the groups v1 builds.

---

## Standing principles

- Write it yourself; skeletons only for new machinery.
- Receipts before claims: read the remote branch, and "works" means a tally.
- Design before code; rules with combining cases go in a spreadsheet first.
- The server is law, the client is courtesy.
- Stores execute, services legislate. Fakes mirror store mechanics, never policy.
- Derived, not stored: facts are fields, answers are functions that take the clock as a parameter.
- Test rigor by tier: full for service rules, permissions and visibility; one happy path for a browser feature; none for markup the server already guards.
- The installed source outranks every mentor, human or AI.