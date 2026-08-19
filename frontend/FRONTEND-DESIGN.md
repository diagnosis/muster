# Muster Frontend — Screen Map & Design Rulings (v0)

The frontend's founding doc, sibling to the backend's engineering reference and
the suite's TEST-DESIGN.md. Every ruling below was argued before code; changes
go through the same door — argue, rule, then build. Rulings carry their
reasoning so future-you can re-litigate honestly instead of guessing.

**Stack (locked):** Vite · React · TypeScript · TanStack Router · TanStack
Query · CSS Modules (plain CSS — Tailwind parked until `display: flex` feels
like boilerplate, not a decision). ESLint. No UI library.

**Dev setup:** Vite proxy forwards `/api` → `http://localhost:8088`; all fetch
paths are relative. CORS is a deploy-day problem, deliberately deferred.

---

## Founding principles

1. **Anonymous-first.** The front door is the content. No gate screen, no
   login wall, no auth bootstrap blocking first paint. Auth lives in the
   header corner and appears as a wall only at write-intent (the join tap,
   hosting). Mirrors the API: reads public, writes authed.
2. **One page per resource, rendered per viewer.** `/outings/:id` serves
   stranger, member, and host from one route and one fetch — sections and the
   action slot swap by viewer state. Viewer never forks pages; *task*
   sometimes does.
3. **Pages and buttons are state renderers.** The UI gives faces to states the
   API already defines and the e2e suite already proves. Debugging is "which
   state did the data claim vs. which face rendered."
4. **Defaults do the work of steps.** Forms interact only where no default is
   honest; each such field gates the submit button, not the navigation.
5. **The server is the law; the client is courtesy.** Client validation is UX;
   every rule is enforced server-side regardless (400/403/409 render as
   states, never as surprises).

---

## Screens

### 1. Outing list — `/` (public)
- List of **cards**, soonest-first (ListUpcoming's own order — no client sort).
- v0 shows all upcoming outings. ("Nearby" is parked: needs coords on create,
  a distance query param, and geolocation UX — three layers, v1.)
- Card contents (list payload only — no seat math at list level in v0):

      ┌─────────────────────────────────┐
      │ Red Mountain                    │  title — biggest
      │ Snoqualmie Pass                 │  destination
      │ Sat Aug 13 · 7:00 AM            │  starts_at, humanized
      │ [hard] [relaxed]     $3.00/seat │  badges + cost
      └─────────────────────────────────┘

- Cost renders **only when nonzero** (0 → nothing, never "$0.00").
- Card = `<OutingCard outing={Outing} />`; card-vs-row is CSS, not
  architecture. Mobile-first: full-width stacked.

### 2. Outing detail — `/outings/:id` (public, viewer-aware)
One page, one fetch (`GET /api/outings/{id}`), sections in order:
title/meta → **action slot** → seat-math strip → host card → roster → notes.

- **Seat-math strip:** the four derived numbers. `seats_short > 0` renders as
  a recruiting pitch, not just an alarm: "⚠️ 2 more seats needed — join as a
  driver?" (allow-and-surface, as product copy).
- **Empty roster** renders honestly ("No members yet — be the first").

**The action slot — eight states, one component** (`my_request` + viewer):

| viewer state            | slot renders                                   |
|-------------------------|------------------------------------------------|
| anonymous               | **Request to join** → auth with context        |
| logged in, no request   | **Request to join** → join form                |
| `requested`             | "Requested — waiting on host" + Withdraw       |
| `accepted`              | "You're going!" + Withdraw (confirm — heavier) |
| `declined`              | terminal message, no action (API 409s re-ask)  |
| `withdrawn`             | **Request to join** again (re-request legal)   |
| host                    | host controls (see below)                      |
| cancelled / past outing | status text only — overrides all rows          |

v0 note: withdrawn → re-request silently keeps the ORIGINAL role/seats
(backend v0 spec, scheduled to change v1-early). UI asks fresh; API decides.

### 3. Join form (reached from the slot; modal or route — decide at build)
Single form. **No wizard** — four fields don't earn one; the forcing function
is the submit gate, not Next buttons.

- **Role:** two big buttons ("🚗 I can drive" / "🎒 I need a ride") — radio
  semantics, no default. The one true gate.
- **Seats offered:** appears only when driver is selected. **Role-change
  resets seats to 0** (no stale payload smuggling).
- **Guests:** stepper (− 0 +), defaults 0 — zero is an answer, not an
  untouched field.
- **Note:** optional textarea ("Anything the host should know?").
- **Submit:** disabled until valid (`role chosen && (rider || seats ≥ 0)`),
  labeled "Request to join." Success lands back on detail — the slot showing
  "Requested" IS the receipt; no toast needed.

### 4. Host view (detail page, host slot + inbox)
- **Slot controls:** Edit · Cancel · Requests entry. The seats_short alarm
  speaks loudest here ("recruit a driver or shrink the party").
- **Cancel:** confirm modal stating stakes ("Members will see it as
  cancelled. This can't be undone."), confirm button says **"Cancel outing"**
  (buttons name their verb).
- **Inbox (RULED, final, round four):** inline **"Requests (N)"** section on
  detail → compact rows (name · role badge · "+1 guest") → **tap opens the
  decision modal**: full note, seats/guests, current seat math restated,
  **Accept / Decline** as the modal's verbs. The capacity 409 renders inside
  the modal as its error state ("Can't accept — needs 3, only 2 spots left").
  Scan inline; decide focused. Escape hatch if hosts ever drown in requests:
  "view all →" page — v1, evidence-triggered.
- **Edit** is a separate destination (form of PATCH semantics — only changed
  fields sent). Editing is cold/rare; request-managing is hot/frequent —
  task frequency decides real estate.

### 5. My outings — `/me/outings` (authed — one of only two guarded routes)
Hosting / joined buckets from `GET /api/me/outings`. Design at build time.

### 6. Auth screens
Login / signup as modals-or-routes reached **from intent** (join tap, host
tap, header button) — never as a lobby. Cookie flow; silent refresh arrives
in the auth chapter when a 401 is first felt.

### 7. Profile
`PATCH /api/me/profile` form + public hiker card. Design at build time.

---

## Architecture rulings

- **Client:** `src/lib/api.ts` — `ApiResponse<T> = {ok:true; data} |
  {ok:false; httpStatus; error}`. Never throws; unwraps the envelope;
  `httpStatus` (HTTP) and `error.status` (app code) are different numbers,
  deliberately both kept. Refresh/retry machinery arrives in the auth
  chapter, WHEN its failure is felt — not before.
- **Types:** transcribed from the wire (curl first), never from Go memory.
  `omitempty` → `?`. The wire grades the homework.
- **Server state belongs to TanStack Query** (chapter 2): the manual
  useEffect version was written once, on purpose, to feel what Query
  replaces.
- **Auth-check-as-cached-query** (inherited from deploywatch review):
  `ensureQueryData(meQueryOptions())` — identity is one cached query.
  **Rejected from deploywatch:** blocking bootstrap before first render
  (anonymous-first forbids it); wall-to-wall route guards (Muster guards ~2
  routes); `instanceof Error` guard dance.
- **Components are typed, single-purpose, extracted at need** — third-use
  rule applies to helpers and hooks alike.

## Build order (chapter plan)

1. ✅ First light: scaffold, proxy, types, api.ts, raw-effect list
2. `OutingCard` + CSS Modules (the designed card)
3. TanStack Query enters (list converts)
4. TanStack Router (file-based; `/` and `/outings/:id`)
5. Detail page + action slot (anonymous states first)
6. Auth chapter: login/signup, me-query, cookie flow, THEN refresh logic
7. Join form + member states
8. Host tooling: controls, cancel modal, inbox + decision modal
9. My outings, profile
10. Polish pass, deploy chapter (CORS decision lives here)

## Parked (with triggers)
- Tailwind — when flex/grid feels like boilerplate
- test.extend fixtures — if teardown/shared-singleton need appears
- Nearby/geolocation — v1, three layers
- Inbox "view all" page — if request volume ever demands it
- Frontend e2e (Playwright component/browser tests) — after screens exist

---
---

# Part II — The Design System (as built)
*Transcribed from source at `feat/ui-ship` @ `03b60df` (2026-08-12). Part I ruled the screens before code; Part II records the visual system that emerged. Arguments settle here, once.*

## 1. The three layers

| Layer | File | Owns |
|---|---|---|
| **Tokens** | `src/index.css` `:root` | Every color, space, type, radius, width value. No raw values below this layer. |
| **Base** | `src/styles/base.css` | Element defaults for `button`, `input`, `textarea`, `select` + the `.btn-primary` global variant. Reset lives in index.css (`box-sizing: border-box`, `* { margin: 0 }`). |
| **Modules** | `*.module.css` beside each component/route | Everything scoped: layout, one-offs. camelCase class names, always. |

**Rule:** components style themselves; containers arrange. A component never imports another's module or a global page stylesheet.

## 2. Color — "alpine lake"

Deep teal over warm stone neutrals. Picked live in devtools against white text; the primary must carry `--color-on-primary` comfortably.

```
--color-primary       #0d7377   brand; primary actions, selected states, focus ring
--color-primary-dark  #0a5d61   hover = same hue one notch darker, never a costume change
--surface-page        #F7F5EF   the page (warm paper)
--surface-raised      #ffffff   cards, panels, inputs
--border-subtle       #E9E3D8   the only border color
--color-text          #3B362B   body
--color-text-muted    #706A5C   meta, secondary
--color-danger        #B3261E   errors only
--color-on-primary    #FFFFFF   text on primary
```

## 3. Typography

- `--font-display` **Bricolage Grotesque** — headings/display only
- `--font-body` **Source Sans 3** — everything else
- Scale (rem / leading): display 2/1.15 · title 1.375/1.25 · body 1/1.55 · label 0.875/1.4 · meta 0.8125/1.4
- **Rule:** font-size always comes from the type scale, never from spacing tokens.

## 4. Spacing

The set is `--space-1, -2, -3, -4, -6, -8` = 0.25 / 0.5 / 0.75 / 1 / 1.5 / 2 rem — deliberately non-contiguous: the numbers name rem-quarters, so 5 and 7 can join later without renumbering.
**Rules:** spacing tokens are for padding/margin/gap only — never font-size, never border-radius. All default margins reset to 0: every space on the page is one we wrote.

## 5. Radii & layout

```
--radius-card 12px · --radius-control 8px · --radius-pill 999px
--width-shell 44rem (page shell) · --width-modal 26rem
```
Mobile-first: 375px base, a single breakpoint at `min-width: 40em` (640px).

**Header/nav (amended — reality overruled the founding "no hamburger" lean):**
mobile gets a hamburger toggle (44px tap target, `aria-expanded`, outside-click
closes via a document listener with cleanup); the panel drops below the bar on
`--surface-raised` with shadow, `z-index: 20`. At ≥40em the toggle hides and the
same links lay out inline — one markup, two faces. Every nav link closes the
panel on tap.

## 6. Buttons

- **Base `button`/`.btn`**: neutral — transparent bg, subtle border, control radius. Hover = raised surface.
- **`.btn-primary`**: the only global variant. Teal, white text, weight 600. Hover darkens.
- **Disabled**: page-surface bg behind a **dashed** border, muted text — reads as *"waiting on you"*, not faded-out. Source order matters: disabled rules sit **below** hover (equal specificity; order stops hover repainting a disabled button).
- Buttons say what they do ("Create outing", not "Submit"). Disabled submits get a *why* hint.

## 7. Forms

- Controlled inputs, always. State lives in the form component; parents pass identity down (`outingId`) and receive events up (`onClose`).
- **Button-radios**: the real `<input type="radio">` stays in the accessibility tree, visually hidden (position: absolute, 1px box, clip-path: inset(50%)) — never display: none, which strips it from the tree and kills arrow-key navigation within the group. The label is the visible segment; selected wears primary. Every group is a `<fieldset>` with a `<legend>` caption and a shared name on all its inputs, so the browser treats them as one control. Focus ring lands on the label via :has(input:focus-visible), with a negative outline-offset because .radioRow clips overflow.
- **No default on choice fields** — choosing is the gate (`Role | null`, submit disabled until chosen). Role-change resets role-owned fields (seats → 0) on *every* radio's onChange.
- `type="email"` / `type="password"` / `datetime-local` → `toISOString()` at submit; money edited in dollars, wired in cents (`Math.round(d*100)`).
- Errors render in `--color-danger` near the action; the server's message is the copy. Client checks are courtesy; **the server is the law**.

## 8. Interaction states

- The action slot is a state renderer (`renderSlot`): cancelled override → anonymous → status ladder → default. One frame, swapping faces; never two truths at once.
- Capacity copy distinguishes **full (cap-bound)** from **seat-bound** ("a driver could open more spots").
- Modal: backdrop click closes, panel stops propagation, `z-index: 10`. Modal owns the shell only — content chrome (the ✕, headers, actions) belongs to the caller; HostControls renders its own `.modalHeader` with the ✕. Decisions keep evidence visible: inbox math restated in-modal; 409s render *inside* the modal (warn-and-allow — the client warning is courtesy, Accept stays tappable, `AcceptIfCapacity` referees).
- `:focus-visible`: 2px primary outline, 2px offset.

## 9. Conventions

- Spelling: **cancelled** (two Ls), everywhere.
- Module classes camelCase; global classes only `.btn` / `.btn-primary`.
- Copy voice: honest and specific ("No members yet — be the first", "Requested — waiting on host").
- Empty states offer the fix (a door: "Not hosting anything yet — create one").

---

## Part I amendments (reality vs founding rulings, recorded 2026-08-12)

Part I is history and stays unedited; where the build diverged, the build won:

1. **Role button labels** — founding spec's "🚗 I can drive / 🎒 I need a ride" shipped as plain **Driver / Rider**. Shorter, symmetric with the other button-radio trios.
2. **Guests stepper** — shipped as a plain `type="number"` input (min 0, max 3), not a custom stepper. A stepper remains polish-ledger, not a ruling.
3. **OutingCard fields** — the card shows title · destination · date/time · difficulty · pace · cost; `meet_label` moved to the detail meta line instead of the card. Cards sell the hike; detail sells the logistics.
4. **The slot's eight rows** — the founding table had eight viewer states; `renderSlot` ships seven branches: cancelled-override, anonymous, host, requested, accepted, declined, and a default that serves both no-request and withdrawn (re-request legal — one face, two inputs). The withdrawn row merged into the default rather than earning its own copy.
5. **Header** — see the §5 amendment above: hamburger exists; the founding "no hamburger" lean is retired.