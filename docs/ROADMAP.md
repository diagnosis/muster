# Muster — Roadmap

*As of 2026-11-12 · repo: github.com/diagnosis/muster · written from the code on `main` (PR #57 merged) and the `feat/pagination` branch, confirmed by Safa. Replaces the 2026-08-07 roadmap.*

*Doc hierarchy: roadmap = why and when · spec = what · engineering notes = how.*

---

## Where things stand

Muster is live at muster.safadev.app. Backend and frontend are on `main`, gated by CI (`ci.yml`) and deployed by `deploy.yml`. Migrations run before the new binary, so schema changes are expand-only.

Test suites: Go unit tests against a fake store, a Playwright API suite (14 spec files) that proves the real SQL and JSON, and a Playwright browser suite (23 spec files). CI runs both against a throwaway Postgres service container; nothing in CI touches the hosted database.

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
| Passwordless login (PR #48, #49) | A second door on the login page: `POST /api/auth/login-code` mails a six-digit code, `POST /api/auth/login-code/verify` exchanges it for the same session a password login mints. The request endpoint answers 200 whether or not the email exists and whether or not a code was sent (30 s resend window), so the response reveals nothing. A code lives 10 minutes, dies on the fifth wrong try, is burned by a resend, and works once; every rejection is the same 401. An unverified account can ask for a code and is marked verified on success, since the code proves the inbox. Codes are hashed with the hiker id so two hikers drawing the same six digits cannot collide on the unique hash column. Login page: password → request → verify modes with a countdown on resend; one browser spec plants a code through the DB helper  Follow-ups (#49): show/hide eye on password fields; the first send starts the resend countdown |
| Waiting list uplift (PR #50) | Waiting and withdrawn rows now carry `role`, `seats_offered`, `guests` and `is_me`; the hiker id stays server-side and the service marks the viewer's own row. The outing page shows the first three names with a count and a "See all" button, and an amber line with the viewer's place ("You're 3rd of 7") when they are in the queue. The modal lists the whole queue in host order: position, driver tag with seats, "+n guests", the viewer's row highlighted in place |
| Signed-out tabs (PR #51) | When a request's refresh fails with 401, the client forgets the user and lands on `/login` — one callback registered at the root, guarded against re-entry. `/api/auth/me` is a probe: a 401 means "guest", with one quiet refresh attempt and never a sign-out. Login refetches `me` and drops every other cached query; logout nulls `me` and drops the rest (`forgetUserData`). Two-tab, expired-access and guest-page browser specs |
| Capacity on the card (PR #52) | The browse and My-outings lists carry `capacity` per outing (`people_count`, `seat_capacity`, `seats_short`, `spots_left`, `waiting_count`), computed by one `capacityOf` that `Detail` and `Update` now share; the store returns raw per-outing sums in one grouped query. The card gains a fill bar and one sentence in four states, in priority order: seats short ("3 seats short, a driver would help"), no seats left under the max ("a driver would open more"), full, open; waiting appended in any state. The second state was found by a fixture, not designed: 3 people in 3 seats with max 8 read "full" and hid that a driver would change everything. Difficulty and pace chips gain quiet labels ("terrain moderate · pace moderate"). `Detail` gains `waiting_count`, public like the other four numbers |
| Live outing pages (PR #54) | One SSE event, `outing.changed {outing_id}`, broadcast to **every** open connection (`Hub.Broadcast`, no rooms, no audience) after each of nine service transitions: Update, Cancel, RequestJoin, EditJoinRequest, Withdraw, Accept, Decline, RemoveMember, RestoreRequest. Comments don't emit. The outing page filters by id and refetches detail + join requests; browse and My-outings refetch their list on any poke. The page dropped its `notification.created` subscription — one signal per concern. Design with numbered decisions in `backend/docs/events/outing-changed.md`. Anonymous viewers still refetch on focus (the stream is authenticated) |
| My outings: Requested (PR #55) | `GET /api/me/outings` gains a third bucket, `requested`: outings the hiker has asked to join and the host hasn't answered. One store query returns accepted and requested rows together with `jr.status` as a 20th column (`scanOuting` takes extra scan targets), and the store splits them; capacity is attached to all three. The page merges the three into one list per tab with a role label: "You asked" (same words past or upcoming — the host never answered, so that's the only fact). Declined and withdrawn appear in no bucket; the API spec asserts that explicitly, which no test did before — this morning's `IN ('accepted','requested')` with no split went green against the whole suite |
| Hiker card (PR #56) | `/hikers/:id`, reached from every name on an outing (host row, roster, waiting summary and modal, the host's request history and decision modal, comment authors, the DM header). `GET /api/hikers/{id}` now requires auth and returns `hiker.PublicProfile` — name, experience, home area, bio, hosted/went-on counts over past non-cancelled outings, member-since — plus `can_dm`, composed in the handler from `message.Service.CanDM` so the Message button only shows when it would work. Never email, gender or verification; the API spec asserts their absence. Own card shows "Edit profile". The profile form gains Home area and About you (the PATCH existed since v0; nothing called it). `ListedHiker` now carries `hiker_id` on the wire. Signup accepted no `home_area` until this PR: `RegisterInput` had no JSON tags behind `DisallowUnknownFields` |
| Sessions per device (PR #57) | `user_refresh_tokens` held one row per hiker per platform (`UNIQUE (hiker_id, platform)`), so logging in on a second browser or a phone deleted the first one's token and it was signed out on its next refresh. Now one row per login; rotation is one `UPDATE … WHERE token_hash = $old AND platform = $p AND expires_at > now() RETURNING …`, atomic by construction; a token rotated in the last 10 s still refreshes (`previous_hash`, `rotated_at`), covering a browser whose refresh response was lost. Logout revokes only the presenting session; password reset still signs out everywhere. Design and a sequence diagram of the grace path in `backend/docs/login/refresh-rotation.md` |
| Browse pagination (PR #58) | `GET /api/outings` returns one page: `{items, next_cursor}`, 20 by default, 50 at most (a limit outside the range is clamped, never refused). Keyset, not offset: the cursor is the `(starts_at, id)` of the last row returned, sent as an opaque base64url string in `?after=`; a cursor the server can't read is a 400. The service asks the store for one row more than the page to learn whether a next page exists. New index `outings (starts_at, id)`: the first page went from a sequential scan and sort of ~10,800 rows (3.3 ms) to an index scan of the rows returned (0.1 ms), and stays there at any depth. The browse page uses `useInfiniteQuery` with a Load more button; `outing.changed` refetches the pages already loaded. First feature whose query was written and `EXPLAIN`ed in DataGrip before it was a Go string. Design in `backend/docs/outing/pagination.md` |

---

## v1 — what is left

In this order. Items can be added.

1. **Pagination, the rest.** Browse shipped in #58. Still open: chat (the messaging design's D5 makes it a v2 trigger at 100 messages per conversation; decide whether it stays in v1) and server-side phase filtering. My outings' Past tab is v3+; until then it gets a safety `LIMIT` in the audit.
2. **Audit.** After the items above: every list endpoint checked for limits, every permission rule checked against its test, error codes made consistent.
3. **One real hike organized through Muster.** v1 is done when boots hit dirt.

### Known bugs

- **`verify-email` shows "expired or already used" on a fresh link in dev.** React StrictMode runs the page's effect twice, the first call consumes the token, the second gets a 409 and overwrites the success message. CI builds (no StrictMode) pass, so the suite is run with `CI=1`. Fix: make the verify call fire once (a ref guard or a mutation keyed on the token).

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
- Host's Restore button still uses `HostControls.module.css .restoreBtn`; `.btn-outline` is global now, use it.
- Collapse the `seedJoinRequest*` test helpers into one with an options struct.
- Bare `tsc` in `frontend/` reports ~98 narrowing errors; `tsc -b` is clean. Find out which config the bare command reads. A frontend `make check` (tsc -b, eslint, build) would make this moot.
- Two `uuid.UUID` parameters side by side (`hikerID, requestID`) were swapped twice this month; typed ids (`type RequestID uuid.UUID`) would make it a compile error.
- Reset-password page should clear the access token and query cache on success, since every session is revoked.
- The prod connection string lives in two places: `/etc/muster/env` on the droplet (pooled, for the server) and the `NEON_PROD_URL` secret (direct, for goose in deploy). One source; the Neon move was slowed by updating one and not the other.
- `make migrate-up` needs the direct (non-pooled) URL; goose's prepared statements fail through PgBouncer. A `MIGRATE_URL` or a note in the Makefile.
- Delete the unused `NEON_API_KEY` and `NEON_PROJECT_ID` repo secrets.
- Hosting: Neon free (100 CU-hours/month, scale-to-zero) is enough with CI off it. When real hikers arrive: a Postgres on the DO droplet never pauses; Supabase free pauses after 7 idle days; Neon Launch is ~$19.
- Login in one tab doesn't reach the others until they reload; a `BroadcastChannel` or `storage` event would. Logout already propagates on the next request.
- The `signingOut` re-entry guard in `api.ts` has no test; the frontend has no unit runner. Decide on vitest for `lib/`, or accept it as untested plumbing.
- `me` staleTime is 5 minutes; a stale identity hid the two-tab bug for a week. 30 s, or `refetchOnWindowFocus` for `me` alone.
- `isAudience` includes the host, `notifyOutingAudience` doesn't: two homes for "who belongs to this outing" that disagree. One `audience(ctx, outing)` helper for both, with a test (host + 2 accepted + 1 pending + 1 declined → 4 ids). Found while designing #54; not fixed there.
- `Hub.Broadcast` sends every outing change to every signed-in tab. Fine at tens of connections; if it ever shows in a profile, the next step is outing rooms in the hub, not audience fan-out.
- Double poke: the subject of a transition gets `notification.created` (bell) and `outing.changed` (page). Two listeners, one refetch each; React Query dedupes. Acceptable, watch the network tab if it ever looks like more.
- A wiring test in `internal/api`: register a client on the real hub, POST a join through the router, read one `outing.changed` from the channel. Twenty lines; the only seam #54 left untested by code.
- Login cross-tab broadcast: same hub, `login` event to the user's other tabs, so a sign-in on one tab lifts the signed-out state on another.
- Logout is behind `requireAuth`; with an expired access token it 401s before reading the refresh cookie, so the session row survives and the cookies stay. The refresh cookie is the credential being revoked; move the route out of the guard.
- Expired refresh-token rows are never deleted now that login no longer clears the platform's row. A weekly `DELETE WHERE expires_at < now() - interval '30 days'` once there's a scheduler.
- Refresh-token families with reuse detection (v2): a superseded token presented after the grace window is a theft signal; revoke every token descended from that login. One `family_id` column on top of #57.
- A "your devices" list on the profile page, now that rows are per session, with "sign out of all devices" calling the existing `DeleteAllRefreshTokens`.
- `wantStatus` in `hiker` tests now fails cleanly on a nil error; `outing`'s copy of the helper may not — check.
- Gender on outings, **by demand only** (v2): three steps in order — "show my gender on my card" opt-in on the profile form (off by default; the card shows it only when on), then an aggregate "N women going" count on the outing page, then women-only outings as a host setting. Never the raw signup field on a public card. Design before code.
- Profile photo or avatar upload: storage, a column, image resizing. The card uses the initial today, same as roster rows.
- Hiker card as a drawer inside the host's decision modal: `ProfileCard` is already its own function so the same component can render in a slide-in; today "View profile" navigates away and the modal is gone on back.
- Outing-page modal state in the URL (`?waiting=1`, `?request=<id>`) so browser back restores an open modal. ~15 lines with TanStack `validateSearch`.
- Clearing home area or bio: PATCH treats a missing field as "leave alone" and can't clear to null (v0 note in `UpdateProfileInput`); the form just doesn't send an emptied field. Needs a sentinel or a separate clear endpoint. Bio has a 500-char client cap and no server limit.
- The waiting-summary name join (", " / " and " / "and N more") is inline JSX on the outing page; if a third list needs it, make it a `JoinedNames` component.
- `.btn-quite` is a typo for quiet, used once in `Comments.tsx`.
- A history view ("I withdrew by mistake, where was that outing?"): withdrawn and declined requests are deliberately absent from My outings. If it's ever asked for, it's a separate list, not a fourth bucket.
- `ListForHiker` runs two queries (hosting, requests) that select the same 19 columns; one `listOutings(ctx, query, args...)` helper would collapse the two scan/iterate blocks. SQL-week exercise: rewrite `ListForHiker` as one query with a `CASE`/`UNION` and compare `EXPLAIN`.
- Browse grid on desktop: tried in #52 and reverted. Two 22rem cards don't fit the 44rem shell, and the chips wrap at that width. If it comes back: widen the shell for list pages only and let width decide the columns; never a per-row setting.
- `CapacityLine` on the detail page too: its "4 going · 1 spot left" block and bar are the same thing in a bigger font. One component, one home.
- `Detail` serves capacity flat (`people_count` at the top level) while the lists nest it under `capacity`. Tolerable; flatten or nest both when the detail page adopts `CapacityLine`.
- SQL study week (from 2026-10-25): `CapacityStats` is day 3's exercise, rewritten from scratch against the dev DB; `EXPLAIN` on `= ANY($1)` is day 7's.
- Bare `tsc` reads a different config from `tsc -b` and reports ~98 narrowing errors (`verify-email.tsx` first); `npx tsc -b` is the one the build and Playwright use.
- The waiting-list summary shows "See all" only past three names, so a queue of two or three never reveals roles or guests; consider showing the button always.
- The "(you)" marker sits beside a name that may truncate on a phone; the amber row carries the meaning, but check it.
- `listNames` and the page both add "and" when a "more" count follows; join with commas when there is a count.
- Login handlers hardcode `PlatformWeb`; the mobile client will need platform in the body or a header, for password and code login alike.
- `"invalid platform"` is written four times in `hiker/service.go`; one `validatePlatform` helper.
- The login page carries the 30 s resend window and "10 minutes" as literals mirrored from the server; a `retry_after` and `expires_in` on the request response would give them one home.
- `mintLoginCode` and the sha256 mirror of `authtoken.hashCode` exist twice, in `backend/e2e/db.ts` and `frontend/e2e/fixtures/db.ts`; the browser suite should import the API suite's helper.
- Password field: a show/hide eye toggle on the login and signup forms.
- A `GET /api/dev/login-code/{hiker}` route in dev/test would let specs read a real minted code instead of planting one; the hash mirror then goes away.
- Two `authtoken` fakes (`authtoken/fake_store_test.go` keyed by id, `hiker/fake_token_store_test.go` keyed by hash); a shared `authtokentest` package was tried and dropped.
- Browse filters, by demand: date range and area first ("what's on this weekend near me"), then host; title search last (needs `pg_trgm` or full-text search, a plain index can't serve `ILIKE '%…%'`). A filter is one more `WHERE` condition and leaves the cursor alone while the order stays `(starts_at, id)`.
- Hike distance and elevation gain on an outing: two nullable columns, fields on create and edit, shown on the card and detail page; then filters on browse. Typed by the host. Decide after the audit whether it goes before the first real hike.
- A Go test for the page-size clamp (limit 0 → 20, 1000 → 50, with 60 seeded). The API spec's "at most 50" proves nothing on a database with fewer than 51 open outings, which is what CI has.
- `backend/e2e/types.ts`: `PageResponse.next_cursor` is typed `?: string`; the server sends `string | null`.
- `Test_ListUpcoming_Pages` comments still say outings 6 and 7 are the tied pair; the code ties 9 and 10.
- `outing.Cursor` has no JSON tags, so the token carries `StartsAt` and `ID` as keys. Short tags would shrink it; changing them invalidates cursors in open tabs for one request.
- `decodeCursor` accepts the base64 of JSON `null` and treats it as the first page. Harmless; a strict check would 400.
- The front of the browse list is shared by every spec that parks an outing there (`moveToFront`, `setOutingTimes` at now + 5 min). `showOnBrowse` copes by clicking Load more. If the API suite ever needs the same, give it a helper that follows `next_cursor` until an id turns up.
- `moveToFront` exists in the browser suite only; the API suite's capacity and pagination specs write the same two lines inline.
- With an infinite query, each `outing.changed` poke refetches every loaded page of browse. Fine at tens of connections; if it shows up, refetch only the first page or patch the changed card.
- `CREATE INDEX` in a migration takes a lock that blocks writes for the build. Instant on today's table; use `CREATE INDEX CONCURRENTLY` (outside a transaction) once a table is large.

---

## v2 — the map, on the web

- Trail and meeting-point map on the outing page and on browse.
- Trail picker on create: choose a trail and pre-fill distance, elevation gain and difficulty. Needs a trail data source (OpenStreetMap has geometry; no public API is known for AllTrails or WTA). Only if hosts say typing two numbers is a problem.
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