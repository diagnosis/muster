# Muster — Roadmap

*As of 2026-10-18 · repo: github.com/diagnosis/muster · written from the code on `main` (PR #47 merged) and confirmed by Safa. Replaces the 2026-08-07 roadmap.*

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
| Waiting and withdrew lists (PR #43) | on the outing page for logged-in viewers, names only; waiting in queue order (drivers first since PR #47, then `requested_at`), withdrew most recent first; the going list is sent by viewer level: count only when logged out, names when logged in, hiker ids for members, request ids for the host |
| Chat closure (PR #44) | outing chat is read-only 7 days after the outing ends, the same window as comments; history stays readable and delete stays allowed; the chat page shows the server's reason when a send is refused |
| Request history and restore (PR #45) | the host's requests list takes `?status=` (requested, declined, removed; anything else is 400); declined and removed requests can be restored to the queue at their original `requested_at`, host-only and only before the outing starts; the hiker gets a `join_request_restored` notification; the outing page shows Waiting, Declined and Removed tabs with a Restore button on history rows |
| Change password (PR #46) | the profile page sends the signed-in user the existing reset link; there is no in-app password form. Completing a reset now revokes every refresh token on every platform, so no session that knew the old password survives. Dev: air live-reload (`make dev`); `main` requires a pull request |
| Edit pending request (PR #47) | `PATCH /api/outings/{id}/requests/me` lets a hiker change role, seats, guests or note on a pending request; nil fields keep their value and the merged result passes the same validation as a new request. An edit that asks the outing for more — more guests, fewer seats offered, or driver → rider — moves the request to the back (`requested_at = now`); anything else keeps its place. The host gets a `join_request_updated` notification on role, seats or guests changes; note-only edits are silent. Drivers now lead the queue everywhere: both the host's list and the outing page's waiting chips order by role, then `requested_at`. The outing page shows what you asked for, an Edit button, and a live warning when the change would move you back |
| Passwordless login (PR #48) | A second door on the login page: `POST /api/auth/login-code` mails a six-digit code, `POST /api/auth/login-code/verify` exchanges it for the same session a password login mints. The request endpoint answers 200 whether or not the email exists and whether or not a code was sent (30 s resend window), so the response reveals nothing. A code lives 10 minutes, dies on the fifth wrong try, is burned by a resend, and works once; every rejection is the same 401. An unverified account can ask for a code and is marked verified on success, since the code proves the inbox. Codes are hashed with the hiker id so two hikers drawing the same six digits cannot collide on the unique hash column. Login page: password → request → verify modes with a countdown on resend; one browser spec plants a code through the DB helper |

---

## v1 — what is left

In this order. Items can be added.

1. **Waiting list uplift on the outing page.** Show three names and a count, with the full ordered list behind a tap; make the viewer's own position obvious.
2. **Public hiker card page.** The API route exists (`GET /api/hikers/{id}`); the page does not. Home area and bio are in the backend and get surfaced here and on the profile form, together with a profile photo or avatar.
3. **Pagination.** Browse first, with keyset pagination. Then chat, then the Past tab. Server-side phase filtering comes with it.
4. **Audit.** After the items above: every list endpoint checked for limits, every permission rule checked against its test, error codes made consistent.
5. **One real hike organized through Muster.** v1 is done when boots hit dirt.

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
- argon2 parameters injectable, small ones in tests: the hiker package takes 19s under -race because every Register and Login hashes for real.
- The DM-kinds migration Down fails if rows of a removed kind exist (the restored-kind migration deletes first; backfill the older one the same way).
- Full UI audit before v1 ships: one filled button per card, hover states on every custom control, button classes reconciled (`btn`, `btn-primary`, module overrides).
- Edit an accepted request: demote to the queue, or try to fit the new seats and keep the seat if it fits.
- `ListJoinRequests` and `WaitingNames` share a filter and an order and differ only in projection; one query, two projections.
- `MyJoinRequest` is a pass-through so the edit handler can resolve "my request" to an id; `EditJoinRequest` could take the outing id instead (strangers would then get 404, not 403).
- Host's Restore button still uses `HostControls.module.css .restoreBtn`; move it to the global `.btn-outline` the Edit button uses.
- Collapse the `seedJoinRequest*` test helpers into one with an options struct.
- Bare `tsc` in `frontend/` reports ~98 narrowing errors; `tsc -b` is clean. Find out which config the bare command reads. A frontend `make check` (tsc -b, eslint, build) would make this moot.
- Two `uuid.UUID` parameters side by side (`hikerID, requestID`) were swapped twice this month; typed ids (`type RequestID uuid.UUID`) would make it a compile error.
- Reset-password page should clear the access token and query cache on success, since every session is revoked.
- Login handlers hardcode `PlatformWeb`; the mobile client will need platform in the body or a header, for password and code login alike.
- `"invalid platform"` is written four times in `hiker/service.go`; one `validatePlatform` helper.
- The login page carries the 30 s resend window and "10 minutes" as literals mirrored from the server; a `retry_after` and `expires_in` on the request response would give them one home.
- `mintLoginCode` and the sha256 mirror of `authtoken.hashCode` exist twice, in `backend/e2e/db.ts` and `frontend/e2e/fixtures/db.ts`; the browser suite should import the API suite's helper.
- Password field: a show/hide eye toggle on the login and signup forms.
- A `GET /api/dev/login-code/{hiker}` route in dev/test would let specs read a real minted code instead of planting one; the hash mirror then goes away.
- "could not delete refresh tokens" 500 shows up as a flake under the full browser suite (117 specs, parallel logins) and never in 60 isolated runs; check pool size and contention on the `refresh_tokens` delete.
- Two `authtoken` fakes (`authtoken/fake_store_test.go` keyed by id, `hiker/fake_token_store_test.go` keyed by hash); a shared `authtokentest` package was tried and dropped.

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