# Refresh-token rotation — one row per session, atomic rotate, short grace

*Design for `fix/refresh-rotation` (PR #57), updated to the shipped code. Lives at `backend/docs/login/refresh-rotation.md`. Decisions numbered so tests cite them.*

## What's actually wrong (two bugs, one root)

The schema says `CONSTRAINT uq_hiker_platform UNIQUE (hiker_id, platform)`: **one refresh token per hiker per platform.** Everything else follows from that.

1. **Two browsers can't both stay signed in.** Chrome and Firefox are both `platform = web`. Logging in on Firefox runs `mintSession`, which deletes Chrome's token to make room for the new row. Chrome's next refresh finds nothing → 401 → signed out. This is the one you keep seeing when testing with two browsers; it isn't a race, it's the v0 policy ("one session per platform") doing what it says.
2. **Rotation is delete-then-insert, two store calls, no transaction.** A request cancelled between them (tab navigates, EventSource reconnect aborted — `context canceled` in the log) leaves no token at all; the browser retries with the one it has, which was just deleted → "unknown refresh token" → signed out. Rarer, same symptom.

The client's single-flight (`refreshSession` in `api.ts`) is fine and stays. It covers one tab; neither bug is about concurrent calls from one tab.

## Decisions

| # | Decision | Why | Rejected |
|---|---|---|---|
| R1 | **One row per session.** Drop `uq_hiker_platform`. Each login inserts a row; each browser, tab-set or device holds its own token. | Fixes bug 1 outright. "One session per platform" was never a product decision, it was a v0 shortcut. | Keep the constraint and treat Chrome vs Firefox as different platforms: there are only `web` and `mobile`, and a second Chrome profile is still `web`. |
| R2 | **Rotate in place with one statement.** `UPDATE user_refresh_tokens SET token_hash = $new, previous_hash = token_hash, rotated_at = now(), expires_at = $exp WHERE token_hash = $old RETURNING hiker_id, platform`. Zero rows → unknown token. | One statement is atomic without a transaction; a cancelled request either rotated or didn't, never half. Fixes bug 2. `RETURNING` also replaces the separate `GetRefreshTokenByHash` lookup. | `BEGIN; DELETE; INSERT; COMMIT` — works, but it's three round trips and the row identity changes for no reason. |
| R3 | **Grace: a token rotated within the last 10 s still refreshes.** `Refresh` first tries R2's `UPDATE … WHERE token_hash = $old`. On zero rows it tries `WHERE previous_hash = $old AND rotated_at > $since`, which rotates *again* (new hash, `previous_hash` = the hash that was current). `since = s.now() - refreshGrace` is computed in the service, so the rule and its clock live together and the store only compares two timestamps. Beyond 10 s, or no match → `InvalidCredentials`. | The lost-update case: the browser never got the new cookie and retries with the old one. 10 s is longer than any retry and shorter than any attack window that matters here. Rotating again (rather than handing back the existing replacement) keeps the invariant "the server never stores a raw token", since the already-minted raw replacement is gone. | Return the same replacement: needs the raw token stored. Longer grace (5 min): the superseded token is a second valid credential for that long. |
| R4 | **Logout deletes the presented token only.** `Logout(ctx, rawToken)` → `DELETE WHERE token_hash = $1`. "Sign out everywhere" stays what password reset does: delete all rows for the hiker. | With one row per session, logging out of Firefox must not log out Chrome. | Keep `Logout(hikerID, platform)`: with R1 that deletes every web session, which is "sign out everywhere" by accident. |
| R5 | **`SaveRefreshToken` is insert-only** (login, verify, login-code paths). `DeleteRefreshTokens(hikerID, platform)` stays for the reset-password path, renamed `DeleteAllRefreshTokens(hikerID)` since platform no longer scopes anything useful. | Two write paths, each with one job. | — |
| R6 | **Expired rows are deleted lazily by the rotate query** (`AND expires_at > now()` in the `WHERE`) **and swept by a weekly `DELETE WHERE expires_at < now() - interval '30 days'`** when there's a scheduler; until then, ledger. | Rows no longer get cleaned by the delete-all; without a sweep the table grows by one row per login forever. | — |

## Migration

```sql
-- +goose Up
ALTER TABLE user_refresh_tokens DROP CONSTRAINT uq_hiker_platform;
ALTER TABLE user_refresh_tokens
    ADD COLUMN previous_hash TEXT,
    ADD COLUMN rotated_at    TIMESTAMPTZ;
CREATE UNIQUE INDEX uq_refresh_token_hash ON user_refresh_tokens (token_hash);
CREATE INDEX ix_refresh_previous_hash ON user_refresh_tokens (previous_hash) WHERE previous_hash IS NOT NULL;
CREATE INDEX ix_refresh_hiker ON user_refresh_tokens (hiker_id);

-- +goose Down
DROP INDEX ix_refresh_hiker;
DROP INDEX ix_refresh_previous_hash;
DROP INDEX uq_refresh_token_hash;
ALTER TABLE user_refresh_tokens DROP COLUMN rotated_at, DROP COLUMN previous_hash;
-- cannot restore uq_hiker_platform if multiple rows exist; delete all tokens first
DELETE FROM user_refresh_tokens;
ALTER TABLE user_refresh_tokens ADD CONSTRAINT uq_hiker_platform UNIQUE (hiker_id, platform);
```

Prod: this migration signs nobody out (rows are kept), and after it two browsers stay signed in.

## Shape (not code)

**Storage** (`hiker.Storage`):
- `RotateRefreshToken(ctx, oldHash, newHash string, platform Platform, expiresAt time.Time) (*RefreshToken, error)` — R2; `NotFound` on zero rows.
- `RotateFromPrevious(ctx, previousHash, newHash string, platform Platform, expiresAt, since time.Time) (*RefreshToken, error)` — R3; same contract. Platform and expiry are `WHERE` conditions on both rotate methods.
- `SaveRefreshToken` unchanged. `DeleteRefreshToken(ctx, hash)` — R4. `DeleteAllRefreshTokens(ctx, hikerID)` — R5.
- `GetRefreshTokenByHash` and `DeleteRefreshTokens(hikerID, platform)` go away; the compiler finds the callers.

**Fake store:** rows keyed by `token_hash`; rotate = delete key, insert new key with `PreviousHash` set and `RotatedAt = now()`; `RotateFromPrevious` scans for `PreviousHash == x && RotatedAt.After(since)`. Mirrors the mechanics; the window is the *service's* rule, arriving as a cutoff time, never the fake's.

**Service:**
- `Refresh(ctx, raw, platform)`: hash → `RotateRefreshToken`; on `NotFound` → `RotateFromPrevious(…, graceWindow)`; on `NotFound` → `InvalidCredentials`. Then expiry and platform checks on the returned row (unchanged), `GetHikerByID`, sign access, return `Session{…, RefreshToken: newRaw}`. `mintSession` no longer deletes.
- `Logout(ctx, raw)` — the handler passes the cookie value. `ResetPassword` → `DeleteAllRefreshTokens`.
- `const refreshGrace = 10 * time.Second` in `hiker`, with the doc comment from R3. `Service.now func() time.Time` (defaults to `time.Now`) is the one clock: it sets `expiresAt` and `since`.

**Handler:** `handleLogout` reads the `refresh_token` cookie and passes it. Clears both cookies as today.

**Client:** nothing. `refreshSession` single-flight stays.

## Tests

| Tier | What | Red first |
|---|---|---|
| Full | `Test_Refresh_TwoSessionsCoexist`: login twice (two "browsers"), refresh A, then refresh B → both 200, two distinct new tokens | yes — the fake's unique-key behaviour (or the delete-all) makes B fail today |
| Full | `Test_Refresh_GraceAcceptsSupersededToken`: T0 → T1; T0 again within grace → T2; T1 (now the previous hash) within grace → T3; T0 a third time → `InvalidCredentials` (two rotations back, neither column matches) | yes |
| Full | `Test_Refresh_GraceExpires`: refresh T0 → T1, then `svc.now` moved `refreshGrace + 1s` ahead, refresh T0 → `InvalidCredentials` | yes — needs the service's `now`, like `message.Service` has |
| Full | `Test_Refresh_UnknownToken`, `Test_Refresh_Expired`, `Test_Refresh_PlatformMismatch`: existing, must stay green | regression |
| Full | `Test_Logout_OnlyThisSession`: two sessions, logout A, refresh B → 200 | yes |
| Full | `Test_ResetPassword_SignsOutEverywhere`: existing, now via `DeleteAllRefreshTokens` | regression |
| API | `auth.spec.ts`: (a) two contexts log in as one user, both refresh, both 200; (b) one context fires `Promise.all([refresh(), refresh()])` with the same cookie → both 200 (the second lands in the grace path); (c) logout in A, refresh in B → 200; (d) reset password → both 401 | (a) and (b) red today |
| Browser | none new; `login-gate` and `auth` specs are the regression | — |

Sabotage: set `refreshGrace = 0` → the grace test and spec (b) go red, nothing else. Swap `previous_hash = token_hash` for `previous_hash = NULL` in the UPDATE → same two red. Remove `DROP CONSTRAINT` from the migration → spec (a) red with a unique-violation 500.

## Out of scope, ledgered

- Token-reuse detection (a superseded token presented *after* grace is a stolen-token signal; revoke the whole family). Needs a `family_id` column. v2.
- The expired-row sweep (R6) once there's a scheduler.
- A "devices" page listing sessions, now that rows are per session.

## Shipped vs designed

- R6's sweep is not built (ledgered). Rows from abandoned sessions stay until expiry and then forever.
- Logout is still behind `requireAuth`: with an expired access token the logout call 401s before the handler can read the refresh cookie, so the row survives and the cookies aren't cleared. Ledgered; move the route out of `requireAuth`, since the refresh cookie is itself the credential being revoked.
- The SQL trap found on the way: `rotated_at > now() - $5` with a bare `time.Duration` parameter made Postgres infer `$5` as `timestamptz` and the comparison `timestamptz > interval`. Passing a `since` timestamp instead removed the arithmetic, the cast and the trap.
