# Browse pagination — keyset pages over `(starts_at, id)`

*Design for `feat/pagination` (PR #58), written to the shipped code. Lives at `backend/docs/outing/pagination.md`. Decisions are numbered so tests and code can cite them.*

## Problem

`GET /api/outings` returned every open future outing in one response. The query was a sequential scan of `outings` followed by a sort, and the response grew with the table. Nothing limited either. The dev database has about 10,800 open future outings; the browse page rendered all of them.

## Decisions

| # | Decision | Why | Rejected |
|---|---|---|---|
| P1 | **Keyset pagination: "the rows after this one".** A page is defined by the last row of the page before it, not by a row count to skip. | The database starts reading at the cursor's position in the index, so page 500 costs the same as page 1. Rows added or removed ahead of the reader don't shift later pages. | `LIMIT n OFFSET m`: Postgres reads and discards `m` rows on every request, and an outing created while someone is reading pushes a row onto the next page, where it appears twice. |
| P2 | **The order is `(starts_at, id)`, and an index matches it.** `CREATE INDEX outings_starts_at_id_idx ON outings (starts_at, id)`. | `starts_at` alone is not a total order: two outings can start at the same instant, and then "after this one" is ambiguous. `id` breaks the tie. With the index, `ORDER BY` needs no sort step. | Ordering by `starts_at` only: a tied pair that straddles a page boundary loses one row or shows one twice. Both were reproduced by sabotage, in Go and against Postgres. |
| P3 | **The cursor is opaque to clients.** The server encodes the `(starts_at, id)` of the last row it returned as JSON, then unpadded URL-safe base64. Clients send the string back unchanged. | The client has no reason to know what a position is made of. The contents can change (a new sort order, a filter fingerprint) without changing the API. Base64url needs no escaping in a query string. | Two visible parameters (`after_starts_at`, `after_id`): the client must format a timestamp correctly, and every future change to the ordering is an API change. |
| P4 | **Request: `GET /api/outings?after=<cursor>&limit=<n>`. Response: `{"items": [...], "next_cursor": "<cursor>" \| null}`.** No `after` means the first page. `next_cursor` is `null` on the last page. | One field tells the client both "is there more" and "how to get it". | A bare array plus a `Link` header: harder to read from `fetch`, and the envelope already exists. |
| P5 | **Page size: default 20, maximum 50. A limit outside the range is clamped, never refused.** Missing, zero, negative or non-numeric → 20. Above 50 → 50. | A client can't ask for the whole table. Clamping keeps a sloppy client working; there is nothing useful a 400 would tell it. | 400 on a bad limit: an error path with no user behind it. |
| P6 | **The service asks the store for `limit + 1` rows.** If the extra row comes back, another page exists; the extra row is dropped and `next_cursor` is built from the last row **kept**. | Knowing whether there is a next page costs one row, not a `count(*)`. The dropped row is exactly the first row of the next page. | A total count: a second query over the whole filtered table on every request, for a number the UI doesn't show. |
| P7 | **A cursor the server can't read is a 400.** Not base64, not JSON, or missing either field → `BadRequest`. | The token only ever comes from a client. A malformed one is the client's mistake, not a server error. | Treating a bad cursor as "first page": a client bug would silently restart the list. |
| P8 | **The store takes `after *Cursor`; `nil` is the first page.** Two query strings share one `SELECT … WHERE` and differ in the tail: the cursor branch adds `AND (o.starts_at, o.id) > ($2, $3)`. | The store executes; it does not know about tokens, defaults or the extra row. A row comparison against `(NULL, NULL)` is never true, so a single query can't serve both cases without an `OR` that risks the index. | One query with `($2 IS NULL OR …)`. |
| P9 | **UI: a "Load more" button, `useInfiniteQuery`.** Pages accumulate under the one `['outings']` key. | A cursor can only say "continue from here", which is what Load more does. The list is soonest first; nobody wants page 37. | Numbered pages: need `OFFSET` and a total count (P1, P6). Infinite scroll: later, if wanted; the button is simpler and a spec can click it. |
| P10 | **Scope: the browse list only.** | It is the only unbounded public list. | My outings and chat in the same PR. My outings' Past tab is v3+; it gets a safety `LIMIT` in the v1 audit. |

## The query

```sql
SELECT o.id, …, c.id AS conversation_id, o.ends_at
FROM outings o LEFT JOIN conversations c ON c.outing_id = o.id
WHERE o.status = 'open' AND o.starts_at > $1          -- $1 = now, from the service
  AND (o.starts_at, o.id) > ($2, $3)                  -- cursor branch only
ORDER BY o.starts_at, o.id
LIMIT $4;
```

`(a, b) > (x, y)` is a row comparison: true when `a > x`, or `a = x` and `b > y`. It is the same rule as the fake store's `later || sameTimeBiggerID`.

Measured on the dev database (about 10,800 open future outings), first page of 10:

| | Plan | Time |
|---|---|---|
| Before the index | Seq Scan over 10,800 rows, then a top-N Sort | 3.3 ms |
| After | Index Scan on `outings_starts_at_id_idx`, 10 rows read | 0.1 ms |

With a cursor, both the `starts_at > now` condition and the row comparison appear in `Index Cond`; `status = 'open'` is a `Filter` on the rows the index returns.

## Layers

| Layer | Owns |
|---|---|
| Handler (`api/outing.go`) | Reads `after` as a string and `limit` as an int (0 when missing or unparsable). Knows nothing else. |
| Service (`outing.ListUpcoming`) | The clock, decoding and validating the cursor (P7), page-size rules (P5), the extra row and `next_cursor` (P6), then capacity and phase as before. |
| Store (`postgres.ListUpcoming`) | The query (P2, P8). Applies `limit` as given. |
| Fake store | Mirrors the store's mechanics: filter, sort by `(StartsAt, ID)`, skip past the cursor, cut to `limit`. No policy. |

## Live updates

`outing.changed` still invalidates `['outings']`. With an infinite query, React Query refetches every page already loaded, in order, each with the cursor returned by the page before it. A hiker who has loaded three pages gets three requests per poke. Acceptable at this scale; noted on the ledger.

## Tests

| Level | Test | Proves |
|---|---|---|
| Go | `Test_ListUpcoming_Pages` | 25 outings, limit 10 → pages of `[10 10 5]`, every outing once, `NextCursor` nil at the end. Outings 9 and 10 share a start time and sit on the first page boundary. |
| Go | `Test_ListUpcoming_SortedOpenFuture`, `_Phase`, `_Capacity` | The first page still filters, sorts, stamps phase and attaches capacity. |
| API | `pagination.spec.ts` › following `next_cursor` | Three outings at one instant, limit 2: the real row comparison and tiebreak, no row skipped or repeated. |
| API | `pagination.spec.ts` › garbage cursor | 400 (P7). |
| API | `pagination.spec.ts` › limit 1000 | 200 with at most 50 items (P5). |
| API | `capacity.spec.ts` › browse list | The paged response still carries `capacity` for a logged-out viewer. |
| Browser | `browse`, `create-outing`, `capacity`, `live` | An outing created by the spec appears on the browse page, loading more pages if needed. |

Sabotages run, each red at the assertion and restored:

- Fake store: drop the id tiebreak from the cursor skip → `[10 10 4]`, 24 of 25 seen, outing 10 missing.
- Real store: `>` → `>=` in the row comparison → 18 rows returned, 10 distinct.

The first sabotage passed at first. The test's tie was at outings 6 and 7, inside page one, where a time-only comparison is never asked the question. Moving the tie to the boundary made it bite.

## How specs find their outing

A spec that creates an outing and looks for it on the browse list can no longer assume it is on the page it fetched. Two helpers:

- `setOutingTimes` / `moveToFront(...ids)` writes `starts_at = now + 5 min` directly, so the outing sorts near the front. The API's 24-hour rule is checked only on create.
- `showOnBrowse(page, title)` (browser suite) clicks Load more until the card is present, up to five times.

Both are needed. The front of the list is shared: within five minutes of a few runs it holds more outings than one page (28 were counted), and the specs that ran last fell off page one.

## Not in this PR

- Filters and search on browse. A filter is one more `WHERE` condition and leaves the cursor alone while the order stays `(starts_at, id)`.
- Pagination for My outings and chat.
- A Go test for the limit clamp. The API spec's `≤ 50` is only meaningful when more than 50 open outings exist, which a fresh CI database may not have.
- `decodeCursor` accepts the base64 of JSON `null` and treats it as the first page. Harmless; a strict check would return 400.