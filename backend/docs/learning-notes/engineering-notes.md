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
