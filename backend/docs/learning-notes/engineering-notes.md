# Muster engineering notes

Oct 6, 2026 · @safa demirkan

## The first-load refetch race

Invalidating a query during its very first fetch does not start a new request, so the page can keep an answer that is older than the change. This caused every flaky live-update test in Muster.

**What happens**

1. A page loads and a query sends its first request. It has no data yet.
2. Something changes on the server (a message is posted, a marker moves).
3. The app calls `invalidateQueries` because of a poke, a stream open, or a mutation's success.
4. TanStack Query sees the query is already fetching and has no data, so it reuses the request in flight.
5. That request was answered before the change. The page shows the old value and nothing fetches again.

When the query already has data, the same call cancels and restarts the fetch. The gap only exists during the first load.

**How it showed up**

- The inbox count did not arrive, or did not clear, about 4 times in 100 runs.
- The delete-message test and the join-request test flaked the same way earlier.
- It never happened by hand, because a person is slower than a test.

**How it was found**

The test logged every request and response for one page, tagged with the run number. In a failing run there was no `GET /api/conversations` after the `204` from mark-read. In every passing run there was one.

**The fix**

`invalidateFresh` in `src/queries.ts`. It checks whether a matching query is on its first fetch, invalidates, and if so invalidates once more after that fetch has finished.

```ts
export async function invalidateFresh(qc: QueryClient, filters?: InvalidateQueryFilters) {
    const firstLoad = qc.getQueryCache().findAll(filters)
        .some(q => q.state.fetchStatus === 'fetching' && q.state.data === undefined)
    await qc.invalidateQueries(filters)
    if (firstLoad) await qc.invalidateQueries(filters)
}
```

Used in the event stream's `onopen`, the header's inbox events, and mark-read's `onSuccess`. Result: 100 of 100 runs passed.

**Rule to keep:** any code that invalidates because the server told it something changed should use `invalidateFresh`, not `invalidateQueries`. The conversation page, the bell and the outing page still use the plain call.

## Live updates: pokes and the stream

The server never sends data over the event stream. It sends a poke that says something changed, and the page refetches. A poke is delivered at most once, with no replay.

- **A poke sent while nobody is connected is lost.** That includes the first moments after a page loads, before the stream has opened.
- **So the page refetches every time the stream opens**, the first open included. An earlier version skipped the first open to save one round of requests, and that left a gap.
- **The server registers the client before it sends the response headers.** Once the browser sees the stream open, pokes will be delivered.
- **Subscriptions that every page needs live in the header.** The inbox count is one example: `InboxEvents` is mounted once, and the inbox page has no copy.
- **A new event type must be added to `EVENT_TYPES`** in `EventProvider.tsx`, or the browser ignores it silently.

Full contract: `backend/docs/messaging/message.md`, sections 3 and 8.

## Read markers: two guards

The service handles beyond, the store handles below.

| Guard | Layer | How | Stops |
| --- | --- | --- | --- |
| Beyond | Service, `MarkRead` | `min(maxSeq, seq)` | a marker pointing past the last message, which would hide future messages |
| Below | Store, `MarkRead` | `GREATEST(stored, incoming)` in the upsert | a late or stale request moving the marker backward |

- Unread is counted at query time: messages after my marker that I did not write. Nothing stores a count.
- No marker means zero, which means from the beginning. That is right for hosts and for both people in a DM.
- A hiker's marker is created when they are accepted, at the conversation's highest `seq` at that moment.
- The browser reports what it has seen only while the tab is visible. A background tab does not mark anything read.

## SQL patterns used

The question to ask of any query: where does each value come from?

| Function | What it does | Example | Result |
| --- | --- | --- | --- |
| `MAX(col)` | highest value of a column across rows | `MAX(seq)` over 3, 7, 5 | 7 |
| `COALESCE(a, b)` | first argument that is not NULL; a fallback, not a comparison | `COALESCE(NULL, 0)` | 0 |
| `GREATEST(a, b)` | larger of two values in one row | `GREATEST(8, 5)` | 8 |
| `<>` | not equal; never use it with NULL, use `IS NULL` | `hiker_id <> $1` | someone else's rows |

**Highest seq, or 0 when empty**

```sql
SELECT COALESCE(MAX(seq), 0) FROM messages WHERE conversation_id = $1
```

**Upsert that never goes backward**

```sql
INSERT INTO conversation_reads (conversation_id, hiker_id, last_read_seq)
VALUES ($1, $2, $3)
    ON CONFLICT (conversation_id, hiker_id) DO UPDATE
                                                   SET last_read_seq = GREATEST(conversation_reads.last_read_seq, EXCLUDED.last_read_seq),
                                                   updated_at    = now()
```

`EXCLUDED` is the row the insert tried to add and was refused. It exists only inside `DO UPDATE`. The table name refers to the row already stored.

**Why one statement and not select-then-write:** two requests at once can both read the old value and then both write. One statement lets the database do the check and the write together.

**Filling a table from other tables**

```sql
INSERT INTO conversation_reads (conversation_id, hiker_id, last_read_seq)
SELECT c.id, jr.hiker_id,
       COALESCE((SELECT MAX(m.seq) FROM messages m WHERE m.conversation_id = c.id), 0)
FROM conversations c
         JOIN join_requests jr ON c.outing_id = jr.outing_id
WHERE c.kind = 'outing' AND jr.status = 'accepted'
    ON CONFLICT DO NOTHING;
```

Check it with counts: rows inserted against rows expected. Muster's backfill gave 2,843 and 2,843.

**Unread count per inbox row**

```sql
( SELECT count(*) FROM messages m
  WHERE m.conversation_id = x.id
    AND m.hiker_id <> $1
    AND m.seq > COALESCE(
          (SELECT r.last_read_seq FROM conversation_reads r
           WHERE r.conversation_id = x.id AND r.hiker_id = $1),
          0)
) AS unread_count
```

The viewer's id does two jobs: it removes their own messages, and it finds their marker. It sits on the outer query so both union arms get it from one place.

**Two writes in one statement**

```sql
WITH accepted AS (
UPDATE join_requests r SET status = 'accepted' ...
    RETURNING r.outing_id, r.hiker_id, r.updated_at
),
marker AS (
INSERT INTO conversation_reads (...)
SELECT c.id, a.hiker_id, ...
    FROM accepted a JOIN conversations c ON c.outing_id = a.outing_id
    ON CONFLICT (conversation_id, hiker_id) DO UPDATE SET ...
        )
SELECT updated_at FROM accepted;
```

`marker` reads from `accepted`. If the update changes nothing, `accepted` is empty and nothing is inserted. No semicolon inside a `WITH` block.

**Small traps met**

- Double quotes mean a column or table name. Strings take single quotes: `'dm'`.
- An alias used in a subquery must be declared: `FROM conversations c`.
- `ON CONFLICT` goes at the end of the insert, after `WHERE`.
- A query that reads fine can still be wrong. Run it in psql before it goes into Go.

## React and CSS rules that caused bugs

- **Hooks run on every render, in the same order.** A hook after an early `return`, or inside an `if`, crashes when the condition changes. When a hook should only run sometimes (only when logged in, once per list row), put it in a small child component that is only rendered then. `InboxRow` and `InboxLink` both exist for this reason.
- **`??` falls back only on null and undefined. `||` falls back on anything falsy, the empty string included.** Use `||` when the server can send `""`.
- **Negating an "and" gives an "or".** `!canAccept(c)` is true for outing chats too, so a DM filter needs its own `c.kind === "dm"` in front.
- **In a CSS module, a bare class name is renamed.** `.panelOpen .btn` never matches the global `btn` class. Write `.panelOpen :global(.btn)`.
- **Menu state is not screen size.** Hiding something with `!open` breaks when the window is resized. Use a media query when the rule is about width.
- **Two media queries must not share a boundary.** `max-width: 40em` and `min-width: 40em` both apply at exactly 640px. Use `39.99em` for the lower one.
- **A `useEffect` with no dependency array runs after every render.**
- **One home per fact.** A click handler copied to three places, or an event subscription in two components, will drift.

## Testing habits

A test proves something only if it can fail. Each habit below came from a test that could not.

- **Break the code on purpose once.** If the test still passes, it is not testing that code. Skipping the Accept click showed an assertion that passed either way.
- **Assert the side effect did not happen, not only the error.** A refused delete must leave the message in the store and send no poke. A rejected request must never reach the service.
- **Check the real thing, not a local copy.** The struct a call returned does not change when the database row does. Ask the store.
- **An absence check passes on a page that has not loaded.** Wait for something that proves the page rendered before asserting that something is missing.
- **Read failure messages for swapped values.** "expected X got Y" with the two reversed sends you the wrong way on the day it fails.
- **Locators match part of a name.** A button named after a person collides with a "Message" button for the same person. Scope to a region (`getByRole('banner')`) or use a more specific name.
- **Prefer `expect(locator).toContainText(...)` to reading text out.** The `expect` form waits and retries.
- **Handler tests use a fake service** and check input parsing and status codes. Rules about members and conversations belong in service tests. Real accounts hitting the real server belong in the API suite.
- **Count messages, not seq.** `seq` is a position in a global sequence, not a count.

**Commands**

```bash
# one test by title, one project, visible browser
npx playwright test -c e2e/playwright.config.ts -g "validate inbox counts" --project=chromium --headed

# find a flake: many repeats, failures count as failures
npx playwright test -c e2e/playwright.config.ts -g "validate inbox counts" --repeat-each=50 --retries=0

# through npm, arguments for the script go after --
CI=1 npm run e2e -- --repeat-each=2 --retries=0
```

`actionTimeout` in the Playwright config caps how long one click waits. The test timeout is a different setting; leave it longer.

## Debugging method and server logs

The flake was solved by one log, after three guesses did not solve it.

1. **Say what the output shows before saying why.** "Element not found" and "text did not match" are different failures with different causes.
2. **State a hypothesis and the experiment that would disprove it.** A one-second wait confirmed a timing problem. It was a diagnostic, not a fix.
3. **When a prediction fails, stop guessing and collect facts.** Log requests and responses for the one page under test.
4. **Tag the lines when tests run in parallel.** Without a tag, two tests' lines mix and cannot be read.

```ts
const tag = test.info().repeatEachIndex
const interesting = (u: string) => u.includes('/api/conversations') || u.includes('/api/events')
page.on('request',  r => { if (interesting(r.url())) console.log(tag, Date.now(), '->', r.method(), r.url()) })
page.on('response', r => { if (interesting(r.url())) console.log(tag, Date.now(), '<-', r.status(), r.url()) })
```

5. **Compare a failing run with a passing one.** The missing line is the finding.

**Server log: "context canceled"**

The browser hung up before the server answered: a tab closed, a page reloaded, a navigation, or a test ending. The handler and the query are fine. It is currently logged as a warning with a 500, which is wrong for a client that left. Open item: detect a cancelled request context and log it quietly.

**Chat app error: `duplicate_human_message_uuid`**

Not from Muster. The chat interface received the same message twice. Refresh and resend.

## Lessons from adding the end time

- **`= NULL` is never true.** In a `CHECK`, `ends_at = NULL OR ends_at > starts_at` accepts every row, because a check only rejects when the result is definitely false, and "unknown or false" is unknown. Use `IS NULL`.
- **`&&` binds tighter than `||`.** `a != nil && a.Before(b) || a.Equal(b)` still calls `a.Equal` when `a` is nil. Prefer one comparison: "before or equal" is `!a.After(b)`.
- **Validate after applying every field.** A check that compared the new start with the old end rejected a valid move of the whole outing. One rule, in one place, run on the outing as it will be saved.
- **"Not sent" and "sent as null" look the same to a pointer.** A patch can't remove a value by omitting it. Removal needs its own explicit flag (`clear_ends_at`).
- **Column lists and scans must agree.** A select with `ends_at` before `conversation_id` and a scan in the other order swapped the two silently. `make check` can't see this; only the API suite runs the real SQL.
- **Compare times as numbers in tests,** not strings: Go and JavaScript format the same instant differently. Print both values, never a bare true or false.
- **Build boundary times by adding hours,** not by naming two clock times. Fourteen days between two local 6 AMs is an hour longer across a daylight-saving change.
- **Test both directions of a rule.** The cases that must be accepted matter as much as the ones that must be rejected.
-
## Lessons from adding outing phases

- **Stored facts are fields; anything derived is a function.** `StartsAt` is a fact. The phase is an answer worked out from facts plus the clock. If a value can go stale without anyone writing to it, don't keep it in a field that code relies on.
- **Pass the clock in.** `PhaseAt(now)` and `DiscussionOpen(now)` take the time as a parameter, so a test can stand on an exact boundary with a fixed date.
- **A field stamped for the client is empty inside the server.** `o.Phase` on an outing fresh from the store is `""`, which quietly compares as "not upcoming". Server rules call the method. There is exactly one assignment to `o.Phase`, inside `stamp`.
- **One home for "everything computed for the client".** `stamp` sets every such field; all five endpoints call it. A second helper per field means five places to forget.
- **The fake must not remember what the database can't.** A fake that keeps the caller's pointer keeps the stamped phase, and a test passes that would fail against Postgres. Store a copy, and clear fields that aren't columns.
- **A slice passed to a function shares its elements; a slice built from it does not.** `for i := range s { s[i].X = … }` reaches the caller. `for _, v := range s` changes a copy. `slices.Concat`, `append`, `Clone` and `copy` all hand back a different slice.
- **A non-pointer `time.Time` can't say "not set".** Left out of a test row it is year 1, not nil, and the row tests something else than its name says.
- **Rows that must pass catch what rows that must fail can't.** A check that rejected everything was caught only by the "upcoming still succeeds" rows.
- **Same status code, different rule.** Conflict is returned for "full", "not pending" and "already started". Give each assertion a fresh request so only one rule can be the reason, and see it red first.
- **A test that re-implements the rule proves nothing.** State the expected answer; don't compute it with the same logic.
- **Guards built on a positive flag need the `!`.** `if o.DiscussionOpen(now) { reject }` and `readOnly = … || is_open` were both written inverted. Where possible carry one positive name end to end (`canComment`) so there is nothing to flip.
- **Tests that depend on a value must state it.** Random fixtures are fine for fields a test ignores. Capacity is not one of them.
- **In React, derived values are not state.** Only what the user chose is state (`tab`). Counts and filtered lists are computed during render. Hooks go above every early return.
- **Read the failure text before changing anything.** `[404]` and `got false, want true` point at different files.

## Lessons from the waiting list and going-list visibility

- **Default to nothing, then add.** `Detail` starts with empty lists and fills them only when the viewer's level allows. A missed case then shows too little, never too much.
- **Order the viewer levels and compare with `<` and `>=`.** One `levelOf` function decides who the viewer is; the rules read `level >= viewerLoggedIn`. Scattered `if host … else if member …` checks let one viewer fall through.
- **`==` on two pointers compares addresses.** Two pointers to equal UUIDs are different. Dereference, and check for nil first: `p == nil || *p != want`. The same mistake in a test makes an assertion that can never fail.
- **A nil slice is `null` in JSON; `[]T{}` is `[]`.** Initialise every list the client will loop over.
- **A query that never selects a column cannot leak it.** The public lists use their own name-only queries and do not reuse the host's full-row query.
- **Placeholders stand for values, not SQL.** `ASC`/`DESC`, column names and table names cannot be `$1`. Two fixed queries beat one built by string concatenation.
- **Every `ORDER BY` needs a tiebreaker.** Real data already had rows sharing a timestamp. Without `, id` the order can change between queries and a spec flakes.
- **`created_at`, `updated_at` and "when this last happened" are three different things.** A row reused on re-request keeps its `created_at`, and `updated_at` moves on any edit. Queue position got its own column, `requested_at`.
- **One column, one clock.** `requested_at` is always set from Go. Mixing Go's clock with the database's `now()` can misorder rows milliseconds apart.
- **An applied migration is frozen.** Goose never re-runs a version, so a new line added to an old file runs on a fresh database and never on production. Add a new file.
- **`ADD COLUMN … NOT NULL` needs a default and usually a backfill.** The default keeps the old binary's inserts working during deploy; the backfill gives existing rows a true value.
- **Prove a backfill with an aggregate.** A `GROUP BY status` with counts is a receipt; 500 pasted rows cut off by the client's page limit are not.
- **SQL in a Go string is prose until it runs.** A missing comma, a missing semicolon and `FROM hiker` all passed `make check`. Only psql and the API suite can see them.
- **A new column must be added to every scan of that table.** A missed `SELECT` returns a zero value silently, and the next write stores it. Assert the field on a read that goes through the database, not on the response of the write.
- **Check `rows.Err()` after the loop.** `rows.Next()` returns false both at the end and on a failure; without the check a broken read looks like a short list.
- **Reuse `err`.** Separate names per call (`werr`, `xerr`) made it possible to return the wrong one, which was nil.
- **The fake mirrors the store's filter and order exactly, tiebreaker included.** Write the contract down as a small table before either side exists. A fake that sorted by a different column kept an order test green for the wrong reason.
- **Seed in a different order than you expect back.** Otherwise a sort that does nothing passes.
- **A length check is not a content check.** Two lists of two names passed with the lists swapped. Assert the names.
- **A "sees nothing" test only means something where the thing would otherwise be there.** Assert the empty response on an outing that has people in both lists.
- **When a rule change turns an old test red, read its name first.** Behaviour gone: delete it. Behaviour still there but the setup assumed the old rule: fix the setup. Test right, code wrong: fix the code. Changing the expected number is none of these.
- **`toBe` compares identity, `toEqual` compares contents.** Comparing whole objects with `toEqual` also fails when an extra field appears, which makes it a leak check.
- **A locator that matches two elements can pass by accident.** "Log in" passed on mobile only because the header link was hidden. Scope the locator to its container.


## Lessons from closing the outing chat

- **A rule gets one home, even across packages.** The chat calls `Outing.DiscussionOpen`, the same function comments use. Comparing dates against seven days inside the message package would have defined the window twice.
- **Give the service the facts in one read.** `OutingForChat` returns status, start and end together, so the status check and the window check see the same row. A partly filled struct is a trap, so the method is named for its purpose and its comment says which fields are set.
- **Swap the plumbing first, then add the rule.** Replacing `OutingStatus` with `OutingForChat` was done with no behaviour change and the package green, before the new check went in. A failure after that could only be the new rule.
- **A pointer can say "doesn't apply"; a plain value can't.** `IsDiscussionOpen *bool` is nil for a DM. A plain `bool` would have told every DM its discussion was closed. Same trap as a non-pointer `time.Time`.
- **On the client, test a nullable flag with `=== false`.** `!flag` is true for `null`, which would have closed every DM.
- **The JSON key comes from the tag, and only a spec that reads the key by name can check it.** A typo in a tag compiles, passes every Go test, and the page silently reads `undefined`.
- **Put the change in the right struct.** Seven compile errors in a package you did not mean to touch means the edit landed in the wrong place. Stop and check before fixing them one by one.
- **The fake needs one home per fact too.** A helper that filled one map while the method read another returned "not found" for every outing.
- **A table column nothing reads is not setup.** A row declared `StatusCancelled` and the loop never applied it; the test passed or failed for a different reason than its name. Apply every column inside the loop, per row, so rows cannot leak into each other.
- **State the boundary as dates.** `closes` is written out as a date in the test, not computed as `ends + 7 days`, so a change to the window in the code cannot quietly change the test with it.
- **Test the instant, not a second past it.** "One second before" and "exactly at" pin `<` against `<=`. "One second after" passes either way.
- **A test born green needs one sabotage.** The delete test and the DM row passed before any code was written for them. Each was proven by making the mistake it exists to catch.
- **Pick the sabotage that matches the mistake.** Running the outing block for a DM would panic, which proves nothing about the assertion. Giving DMs a default value is the realistic mistake, and it is the one the row catches.
- **To test a rule about the past against the real database, move the row.** The API refuses past times, so the e2e helper `setOutingTimes` updates both columns in one statement, after everything that needs an upcoming outing has been done.
- **After a write in psql, read the result line.** A syntax error and `UPDATE 0` both look like "I ran it". A `SELECT` that returns the computed value (`closes_at`, `closed`) is a better check than reading raw columns.
- **Check which row you changed.** The update went to one outing and the browser was open on another.
- **The client learns a time-based flag when it fetches.** A page left open across the close instant still shows the input; the server refuses the send. The server is the rule and the page is a courtesy, but the page must show the refusal.
- **Assert on the page with `await expect(locator)`.** `expect(await locator.isDisabled())` reads once and fails with a bare boolean. The retrying form waits and names the element.
- **Read the remote, not the message.** "Pushed" was an empty file: `git add .` from a subfolder staged only what was beneath it. Run `git status` from the repo root before committing.

## Lessons from request history and restore

- **Restore is a status change, not a new request.** `SetJoinRequestStatus(..., requested)` touches `status` and `updated_at` only, so `requested_at` survives and the hiker returns to their old place. Reusing the re-request path would have stamped a new time and sent them to the back.
- **Declined and removed are terminal for the hiker, not for the host.** `TestRequestJoin_DeclinedIsTerminal` still passes; only `RestoreRequest` reopens them. One sentence, two rules.
- **Call the guard, don't copy it.** The host, closed and started checks were pasted into `RestoreRequest` and drifted on the first day: a stranger got "this outing is closed" instead of 403. `loadForHostAction` is the one home, and when it was sabotaged every host action went red together, which is the proof it is one home.
- **Permission before state.** The host check runs first so a non-host learns nothing about the outing or the request.
- **A new enum value has more than one home.** `join_request_restored` had to land in the Go constant, `Kind.Valid()`, the CHECK constraint, and `contentFor`. The grep for the previous kind (`dm_reopened`) is the checklist; the `contentFor: unhandled kind` warning was the one that was missed.
- **The migration string is frozen on apply.** `join_request_restore` (no `d`) was in the file for one round. Compare the SQL literal to the Go constant character by character before `goose up`; no unit test can catch it, only a real insert.
- **A CHECK is validated against every existing row.** A Down that puts the old constraint back fails if rows of the new kind exist, so the Down deletes them first. The DM-kinds migration has the same gap and is on the ledger.
- **`notify` swallows its own failure.** A rejected insert logs a warning and the action still returns 200, so a passing restore proves nothing about the notification. The receipt is the `notification_events` row.
- **Validate the filter in the service, and prove the proof.** `?status=banana` reaches the store as a query for a status that never exists: an empty 200 for a typo. `Valid()` already existed, so the test was born green and was sabotaged by removing the check.
- **A helper that builds `?status=undefined` is a client bug the server should refuse.** It did, with 400, which is how the e2e helper's missing guard was found.
- **The query key is the cache key.** Three tabs sharing `['outing-join-requests', id]` show one list. The status belongs in the key, the same way it belongs in the URL.
- **`&&` in an order check only fails when every position is wrong.** `jrs[0] != a && jrs[1] != b` passes `[a, c]`. Compare the ID slice in one `toEqual`.
- **A global `button:hover` outranks a module class of equal specificity by source order.** The active tab went white on hover; the fix is a hover rule in the module.
- **A new word on a page can break an old test on another page.** `getByText(/waiting/i)` in the DM spec matched the new Waiting tab during navigation. Assert the exact copy that only the target page has.

## Lessons from change password

- **Reuse the flow that already proves the right thing.** The emailed reset link proves control of the inbox; an in-app form would have added a second password rule to maintain and a second thing to attack. The profile button calls the forgot endpoint with the user's own email and nothing else is new.
- **Read what the existing flow actually does before trusting it.** "Reset revokes sessions" was assumed; the code did not. The red test (`got 2` sessions after reset) was the proof that the gap was real.
- **Revocation is a security step, not a courtesy.** A failed email is logged and the reset continues; a failed revocation returns the error, because a reset that leaves an attacker's session alive did not do its job.
- **A list of enum values has one home.** `AllPlatforms` feeds `Valid()` and the revocation loop. The first draft listed the three platforms in the method, which is the same drift as the pasted host guard in PR #45.
- **A helper that reads `sent[0]` works until the second user.** `registerAndVerify` always read the first email, so the second hiker's verification consumed the first hiker's used token. Read the last mail, so one user and many behave the same.
- **A test that is born green needs a sabotage that reaches the assertion.** Dropping the fake's hiker filter failed at setup (Login's own cleanup), before the reset ran, so it proved nothing about the assertion. Some fakes can't be sabotaged cleanly; the API spec against Postgres carried that proof instead.
- **See the API spec red against the old binary.** The first run passed because the server was still the pre-change build. A restart turned it red, the fix turned it green; that is the whole point of running it before the code. Air now rebuilds on save so a stale binary is harder to get.
- **The rate limiter sees the whole test suite as one client.** Every Playwright worker comes from 127.0.0.1, so the suite crossed the burst threshold when it grew by one spec and five tests went flaky with 429s. Dev and CI run with high limits from env; prod keeps its own.
- **`go install` for tools, never `go get -tool`.** The second one put air and its Hugo dependency tree into `go.mod`. The Makefile calls the binary by its GOPATH path, so JetBrains' `air` on PATH does not shadow it.
- **A direct push to main is a tooling problem, not a discipline problem.** The branch ruleset refuses it now; a reverted commit and a cherry-pick were the cost of learning that.

## Lessons from edit pending request

- **Rule the feature in a spreadsheet before the first test.** Nine rules, nineteen rows, five sabotages; the table test was then transcription, and "driver → rider restamps" was settled on paper rather than argued over a red.
- **A change asks for more or it doesn't.** More guests, fewer seats offered, or driver → rider asks the outing for more and goes to the back; everything else keeps its place. One `askMore` boolean and one `time.Now()`; the first draft had three stamps in three branches.
- **Merge first, validate second.** `EditJoinRequest` overlays the non-nil fields onto the stored request and then runs the merged result through `validateJoinRequest`, the same function `RequestJoin` uses. Validating the input fields one by one gave a rider sending `seats_offered: 0` a 400 and let a driver who omitted `role` be treated as a rider.
- **Validating before the merge validates yesterday.** The extracted validator was first called on the stored values and every inline check stayed in place to do the real work. Order of operations is the refactor.
- **Pointer fields are a PATCH convention, not a free lunch.** `*int` tells 0 from absent, which the UI never needs today; the cost is a merge step and a `ptr` helper in tests. Kept because a future client that sends one field is plausible.
- **Table rows spell out the full state.** `edit: in{guest: 1}` left `role` as `""` and "note only" was a guest change in disguise. Zero values mean zero; the row says every field.
- **One hiker and one outing per row.** Eleven requests from one hiker on one outing is cross-row state; row 7 can pass because of row 3. `uuid.New()` inside `t.Run` costs nothing.
- **Count assertions must fail in both directions.** `if notify && len != 1` never fails on the rows that exist to prove silence. Compare against the wanted count on every row, and check the kind when it is 1.
- **Capture `before` as a value.** `before := *jr` cannot alias the store; comparing `jr.RequestedAt` to itself after the service mutates it is a test that cannot fail.
- **A `!= nil` guard on an assertion means it only runs when the value exists.** The note check skipped the exact case it was written for (the note never saved, so `Note` stayed nil). Read the pointer into a plain string, then compare unconditionally.
- **A fix without a red is indistinguishable from luck.** The note-only bug was fixed before the assertion existed; the sabotage (put `Update` back inside the `if changed`) was how the assertion earned its place.
- **Inverted condition and inverted test shake hands.** `StartsAt.After(now)` for "has started" plus a test seeding `+1h` and expecting 409 were both wrong and both green. The time check has one home (`validateJoinRequest`); the test seeds a past start.
- **Persist, then notify.** A notification sent before the write tells the host about an edit that may not have landed.
- **The second query is the one you forget.** `ListJoinRequests` got drivers-first; `WaitingNames` kept time order, so the host's tabs and the page's chips disagreed. Found by asking which query the chips read; proven by the API spec reading both as different users, and by reverting the store line to see the names flip.
- **Operand swap, second month running.** `(hikerID, requestID)` called as `(jr.ID, h1)` is a 404 that looks like a fake-store bug. Read the signature before the fake.
- **The name said driver, the body didn't.** `d1` requested as a rider because the helper's default body was used. A variable name is not test data.
- **Playwright's `exact: true` is the plural test.** `"bringing 1 guest"` as a substring passes against "1 guests"; the exact match is what proves the singular.
- **A webServer exit code 2 is TypeScript.** Run `npx tsc -b` on its own and read the first error; Playwright hides it. And a bare `tsc` reads a different config than `tsc -b` — don't fix the code to please the wrong compiler.
- **Say what you have before saying what to do with it.** "Not needed" meant two different things to two people and cost a revert. When the mentor says "use the existing one", ask which one.

## Lessons from passwordless login

- **The mail is the receipt the user gets, so the test reads the mail.** The hiker-side test pulls the six digits out of the rendered body with a regexp and feeds them to the login. That is how a zero `LoginCodeTTL` was found: the body said "expires in 0 minutes" while every store assertion was green.
- **A field added to a config struct is a field every constructor site forgot.** `newTestService` never set the new TTL, nothing said so, and the code was dead on arrival. `NewService` now returns an error for a zero TTL; main's env loader keeps the only default. Constructor validates, startup decides.
- **A silent default in the constructor is two homes.** `if cfg.TTL == 0 { cfg.TTL = 10m }` fixed the test and put the ten minutes in `NewService` and in `getEnvDuration`. Loud over kind for configuration.
- **One answer for every rejection.** `ErrBadCode` covers missing, expired, wrong, burned and unknown-email; the request endpoint answers 200 for unknown and for too-soon. The tests for those cases exist to prove the caller cannot tell them apart, so they assert the same status and the same text.
- **Neutral is about the account, not the outage.** The handler was first written to swallow every error behind the 200. A store failure is a 500 and gets reported; the service already returns nil for the cases that must stay quiet.
- **Hash with the owner's id when the hash column is unique.** Six digits across many hikers will collide; `sha256(hikerID + code)` cannot. Lookup is then by hiker, not by hash, which is also what makes "code for h1 doesn't log in h2" true by construction.
- **A fake that returns its stored pointer hides a missing write.** `tok.Attempts++` on the fetched token "persisted" in the fake and the fifth wrong try never burned the code. The fake now returns a copy and the service calls `IncrementAttempts`; the test then read the stored count, not the local variable.
- **`var Err error` is a nil sentinel.** `errors.Is(err, ErrTooSoon)` with both nil is true, so the throttle "worked" while minting every time. `errors.New` or nothing.
- **A count that includes setup residue fails for the wrong reason.** `len(ft.tokens) != 1` counted the verification token registration left behind. Count the thing the rule is about — live tokens of this purpose for this hiker — and the test stops knowing how registration works. The same line was deleted four times before it stayed deleted.
- **Message and condition must agree.** `if len != 1 { "expected 0" }` lies in one direction or the other. When they drift, one of them is the bug.
- **Order inside the method is the design; a test pins each edge.** Platform before consume (a typo'd platform must not spend the code), consume before verify (an unverified hiker is verified only by a correct code), lookup before any token work. Moving the platform check below the consume turned exactly one test red.
- **Prove the SQL you wrote is the SQL that runs.** Dropping `ORDER BY created_at DESC` from `LiveByHiker` let the older of two live codes log in; the API spec "newest live code wins" went red on a restarted server. A helper that plants a code straight into the table mirrors the Go hash with a comment saying so; the first drift will be a red, not a mystery.
- **Name the spec for what it proves.** "Generate code twice" read as the resend-burns rule, but a planted row burns nothing; what the spec showed was ordering and single use. The burn is proven in Go, where the real `MintCode` runs.
- **The client's throttle is a courtesy, not a clock.** A local 30 s countdown from the request's return is always a hair later than the server's window, so a click that passes the client passes the server. No stream, no server shaving. The first draft forgot to start the countdown on the first send, so the button offered a resend the server would silently drop.
- **Three modes, three rows.** Password, request, verify: what the form shows, what submit does, what the footer offers. Writing the rows first made the component a transcription; the drafts that skipped it had a mode with no branch and a branch with no mode.

## Lessons from the waiting list uplift

- **"Is this me" is a visibility rule, so the service answers it.** The store returns the hiker id in a `json:"-"` field; the service compares it to the viewer and sets `is_me`; the id never reaches the client. Same shape as the roster loop that clears ids for non-members. A first draft wanted the store to take the viewer — stores execute, services legislate.
- **A one-line rule still gets the full treatment.** `IsMe` was one loop, and the Go test caught the copy-paste that ranged `Withdrawn` while indexing `Waiting` — an index-out-of-range waiting for the first outing with more withdrawals than waiters.
- **`w.IsMe != want` fails in both directions on every row.** Three drafts had a check that fired when the row was right, then a second check that fired on every other row. One comparison against a computed `want` replaced a count and two `if`s.
- **A field count that includes setup residue is a test of setup.** `len(ft.tokens) != 2` in the login tests and `len(detail.Waiting)` here both wanted to know how registration or seeding behaves. Count the thing the rule names.
- **Alias once, use everywhere.** `SELECT r.hiker_id … JOIN join_requests jr` is a runtime error Postgres gives and the fake never will; and five columns selected with six scanned is the next one. The API spec with a driver carrying 3 seats and a rider carrying 2 guests is what proves the columns land in the right fields — all-default riders would pass with the columns swapped.
- **Change the shape, expect the old spec to shout.** `toEqual([{name}])` on the waiting list went red the moment rows grew fields. `toMatchObject` for specs about order; `toEqual` on one full row for the spec about the fields.
- **Scoped class names don't reach global classes.** `.waitingSummary .btn-outline` in a CSS module is hashed and matches nothing; `:global(.btn-outline)`. And `styles.notice` from the wrong module is `undefined` without a word from the compiler — the amber line rendered unstyled until the screenshot.
- **Don't wrap a column in a row.** The summary and the amber line became flex siblings inside `.memberRow`; a `.waitingBlock` column with the row's horizontal padding is what the card needed.
- **Mockup first paid for itself twice.** The canvas settled modal-vs-expand and highlight-vs-pin before any code, and the screenshots were checked against it rather than against taste.
- **Live updates have an audience, and it isn't the room.** Pokes go to notification recipients; a fellow waiter gets no notification when someone ahead withdraws, so their "4th of 5" stays until focus. The fix is an `outing.changed` broadcast, filed as v1 item 2.
- **Scope creep announces itself as "while I'm here".** Capacity on the card is a real need and a separate PR with its own list-endpoint work. PR #47 grew into a monster the same way.
- **CI should never spend production's quota.** Neon's free compute went to twenty PR runs, archived `main`, and took the Neon branch action with it. A `postgres:16` service container in the job costs nothing and works on any host; the hosted DB is for prod only.
- **Two homes for one secret is an outage waiting for a rename.** The server's `DATABASE_URL` and the deploy's `NEON_PROD_URL` point at the same database by hand; moving projects meant finding both. Migrations also need the direct endpoint — goose's prepared statements die behind PgBouncer.
## Lessons from the signed-out tab

- **Kill the theory with a receipt before fixing it.** "Refresh token expiry" was the first guess; a 15-second access TTL proved refresh worked fine. The real cause was the cookie disappearing from another tab, and only the two-tab repro showed it.
- **A client needs an opinion about failing to refresh.** `apiClient` refreshed and retried on 401, and when the refresh itself returned 401 it handed the error to whichever component asked. The header kept its cached user; the page showed "Unauthorized" in red; three queries, two identities. One `onSignedOut` callback at the root, fired from `request` when refresh fails with 401, is the whole fix.
- **`/me` is a question, not a session.** A guest's `GET /me` → 401 is the normal answer. Routing it through the refresh path made every guest trigger refresh → 401 → sign-out → clear → refetch `me` → 401 → … fifteen times a second. A `probe` that never refreshes or signs out, plus one quiet refresh attempt inside `getMe` for the expired-but-recoverable case.
- **`queryClient.clear()` is a hammer.** It cancels in-flight queries under mounted components and tripped the error boundary with `CancelledError`. `setQueryData(['me'], null)` plus `removeQueries` for everything else gets the same result without cancelling anything.
- **Login and logout are different identity changes.** Login: drop the guest's cache, refetch `me`. Logout: drop the cache, set `me` to null. The first draft copy-pasted the logout shape into login and blanked the header on every sign-in. `forgetUserData` is the shared half; the `me` line is the half that differs.
- **Refresh only when it's a 401.** A refresh that returns 0 (server unreachable) or 500 is not "you're logged out"; signing people out during a deploy blip would be its own bug.
- **Guard the callback.** Two queries that 401 in the same tick both reach the sign-out branch; a `signingOut` flag makes the second a no-op. Untested today — the frontend has no unit runner — and in the ledger.
- **Fixing the probe broke the reload.** `successful refresh` went red: with `me` as a pure probe, a fresh page load with an expired access token had nothing that refreshed. The browser suite caught it in the same run that proved the fix. Every auth path that the client can be in now has a spec: expired access with a good refresh, failed refresh, logout in another tab, guest on a public page.
- **Headless pages don't get focus.** `window.dispatchEvent(new Event('focus'))` does nothing when `document.visibilityState` is `hidden`, which it is for a background page in Playwright. `reload()` is the honest trigger.