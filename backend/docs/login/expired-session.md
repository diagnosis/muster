# Expired session on a public route

*Design for `fix/expired-session-on-public-routes` (PR #62), written to the shipped code. Lives at `backend/docs/login/expired-session.md`. Read with `refresh-rotation.md`.*

## The bug

A logged-in member left an outing page open. Later the page showed the logged-out view: "Request to join" for someone already going, a roster with only the host, no Discussion. The header still showed them logged in. A reload fixed it.

Reproduced on demand: delete the `access_token` cookie, switch tabs and come back.

## Why it happened

Three facts that are each reasonable and together produce the bug:

1. The `access_token` cookie has the same lifetime as the token inside it. When the token expires, the browser deletes the cookie. An expired session and a visitor who never logged in send the same thing: no access cookie.
2. `GET /api/outings/{id}` allows anonymous viewers. With no access cookie it returned **200 with the public view**.
3. The web client refreshes the session and retries only on a **401**.

When the tab refocused, its requests went out together with no access cookie. The ones that require login (comments, notifications, conversations) got 401, the client refreshed once, and they were retried. The outing request got a 200, was treated as a success, and its anonymous result was cached. The session was repaired within a moment and the page was not.

The cookies are `HttpOnly`, so the page cannot inspect its own session. Only the server can tell it the session is stale.

## Decisions

| # | Decision | Why |
|---|---|---|
| S1 | **On a route that allows anonymous viewers, a request with no valid access token but a `refresh_token` cookie is answered 401.** | The refresh cookie is the evidence that this is an expired session and not a stranger. A 401 is the signal the client already acts on. |
| S2 | **The route only checks that the refresh cookie is present. It does not validate it.** | Looking up, expiring and rotating refresh tokens is `POST /api/auth/refresh`. One home per rule. |
| S3 | **No cookies at all is still a stranger: 200, public view.** | Shared outing links must work for people without accounts. |
| S4 | **When refresh refuses a token, the response clears both cookies.** | Without this, S1 locks out a visitor holding a dead refresh cookie (a session ended elsewhere): 401, failed refresh, login page, on every public outing, until they log in. With the cookies gone, their next request is a stranger's. |
| S5 | **Cookies are cleared only on a refusal (`InvalidCredentials`), not on a server error.** | If the database is briefly unreachable the session is still good, and clearing it would log people out for the server's problem. |
| S6 | **The fix is on the server, not in the web client.** | The client can't see `HttpOnly` cookies. Any other client gets the right behaviour for free, and the rule is testable with a plain HTTP request. |

| Access token | Refresh cookie | Answer |
|---|---|---|
| Valid | any | 200, the viewer's own view |
| Missing or invalid | absent | 200, public view |
| Missing or invalid | present | 401 |

## Code

- `api/server.go`: `refreshIfSessionExpired`, a middleware inside `OptionalAuth` on `GET /api/outings/{id}`. User in context: continue. No refresh cookie: continue as a stranger. Otherwise 401.
- `api/auth.go`: `handleRefresh` calls `clearSessionCookies` when `Refresh` returns `InvalidCredentials`.

The browse list (`GET /api/outings`) returns the same data to everyone and needs no gate. Detail is the only route using optional auth.

## Tests

`backend/e2e/tests/detail.spec.ts`:

| Case | Expect |
|---|---|
| No cookies | 200, `my_request` absent |
| Member with both cookies | 200, `my_request.status` is `accepted` |
| Member's refresh cookie only | 401 |
| Refresh cookie whose session was ended by logout: detail, then refresh, then detail | 401, 401, then 200 with the public view |

The third and fourth were written first and were red: the third returned 200, the fourth failed at its last step.

Checked by hand in the browser: after deleting `access_token` and refocusing the tab, the page keeps the member view, and the console shows a 401 on the outing request followed by the retry.

No browser spec. One would have to fake a tab refocus after removing the cookie; the API spec pins the contract.

## Known gap: three tabs refreshing at once

When a laptop wakes, every Muster tab reconnects its event stream and calls refresh with the same token. The first rotates it, the second is accepted by the 10-second grace window (see `refresh-rotation.md`), and a third presents a token two rotations back and is refused.

That refusal predates this change: the third tab was sent to the login page while the others carried on. With S4 the refusal also clears the cookies, and if its response arrives last, every tab is logged out. One or two tabs are unaffected.

Fix, next on the roadmap: wrap the client's refresh in a Web Locks lock (`navigator.locks.request`), so tabs take turns and each refreshes with the cookie the previous one received.

## Related

- Logout sits behind `requireAuth`, so with an expired access token it answers 401 before reading the refresh cookie (already on the ledger).
