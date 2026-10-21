# Muster — Passwordless login (v1)

Status: accepted · Date: 2026-10-20 · PR #48

## Goal

Let a hiker sign in with a six-digit code sent to their email, as a second door beside the password. Same session, same cookies, same account. Nothing about the account is revealed by asking for a code.

## Rules

| # | Rule |
|---|------|
| R1 | Any registered email can ask for a code; the request always answers 200 with the same message. |
| R2 | An unknown email answers the same 200 and sends nothing. |
| R3 | The code is six digits, hashed with the hiker id (`sha256(hikerID + code)`) so two hikers drawing the same digits never collide on the unique hash column. Lookup is by hiker and purpose, never by hash. |
| R4 | Every rejected code — missing, expired, wrong, burned, unknown email — is the same 401, `code is wrong or expired`. |
| R5 | Asking again burns the live code and mints a new one. |
| R6 | Asking again inside 30 s of the last code mints nothing and still answers 200. |
| R7 | A wrong code counts an attempt; the fifth wrong try burns the code. |
| R8 | A code works once. |
| R9 | An unverified account can ask for a code; a correct code marks it verified, since it proves the inbox. |
| R10 | Password login keeps working after a code login. |
| R11 | An invalid platform is a 400 before any lookup or consume. |

Expiry is 10 minutes (`APP_LOGIN_CODE_TTL`, default in main's env loader; `hiker.NewService` refuses zero). The resend window and the attempt cap are constants in `authtoken`.

## Where each rule lives

- `authtoken.MintCode` — R3, R5, R6 (returns `ErrTooSoon`), creates the row with `attempts = 0`.
- `authtoken.ConsumeCode` — R4, R7, R8; one `ErrBadCode` for every failure.
- `hiker.RequestLoginCode` — R1, R2, swallows `ErrTooSoon` to nil, renders and sends the mail (send failure is logged, the token stands).
- `hiker.LoginWithCode` — R11 first, then lookup (unknown → `ErrBadCode`), then consume, then R9, then `mintSession` (R10 falls out of sharing it).
- Postgres `LiveByHiker` — newest unused token of a purpose for a hiker; expiry is the service's call. `IncrementAttempts` — `attempts + 1` by id.

## Storage

`auth_tokens` gains `attempts INT NOT NULL DEFAULT 0`. Purpose `login_code` already existed in the CHECK. Nothing else is new.

## API

    POST /api/auth/login-code          {email}        → 200 {message}  (400 on empty email)
    POST /api/auth/login-code/verify   {email, code}  → 200 hiker + session cookies, as /api/auth/login
                                                        401 on any rejected code

## Client

One login route, three modes: `password` (email + password), `request` (email only, submit sends the code), `verify` (email read-only, code field, submit logs in). A local 30 s countdown gates "Send again" after each successful request; it mirrors R6 and is always a little later than the server's own window.

## Tests

- Go, `authtoken`: collision (R3), resend burns (R5), throttle at 25 s / 31 s with an injected clock (R6), wrong ×1/×4/×5 (R7), expired (R4), twice (R8).
- Go, `hiker`: request for known / unknown / unverified / too soon; login happy, unverified becomes verified, h1's code against h2, invalid platform leaves the code usable, password after code, unknown email.
- API: happy, unknown email 200, wrong then right (`attempts == 1` read from the table), unverified → `verified_at` set, newest live code wins and a code works once. Sabotage: dropping `ORDER BY` from `LiveByHiker` turns the last one red.
- Browser: one path — request from the page, plant a code through the DB helper, type it, land signed in.

## Out of scope (v1)

Magic links · platform from the request body (both login handlers hardcode web) · `retry_after` / `expires_in` in the request response · a dev route that returns a real minted code for specs.