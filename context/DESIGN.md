# Spotfile — Design System

Source of truth for tokens. Adapted from the Note (Flutter) design system: token
values, rules and rationale are unchanged; only the implementation mapping is
translated to Spotfile's stack (Wails webview + Svelte + TypeScript + CSS).
Examples that mention classes, courses or the calendar come from Note's domain —
apply the same rule to Spotfile's equivalents.

Every value here maps to a **CSS custom property** defined once in
`frontend/src/styles/tokens.css`. Components read tokens through `var(--…)` —
never through raw hex, raw `px` spacing, or ad-hoc `font` declarations in a
component's `<style>`.

**Naming:** a camelCase token becomes a kebab-case property with its group as
prefix — `surfaceRaised` → `--color-surface-raised`, `lg` → `--space-lg`,
`rSm` → `--radius-sm`, `body` → `--text-body-*`, `quick` → `--motion-quick`.

**Themes:** Nocturne and Daylight are two sets of `--color-*` values, selected by
`:root[data-theme="nocturne" | "daylight"]`, defaulting from
`prefers-color-scheme`. Nothing outside `tokens.css` names a hex value.

> **Status:** implemented in `frontend/src/styles/tokens.css` (tokens, both
> themes, reduced motion) and `styles/fonts.css`, and used by the app shell,
> search view and previews. `tokens.test.ts` enforces the contrast floor.
> Not used yet: the class identity ramp and the display entrance choreography.

---

## 1. Direction

**Nocturne Study.** A desk in a dark study at night: a cool ink-blue room, and
one warm lamp falling on the work surface. Two ideas follow from that image and
run through the whole system — *light falls on one thing at a time* (elements
sit on one of three planes, and only the focused plane is lit), and *the lamp is
the warm thing in a cool room* (so warmth means action, and everything cool
recedes into reference).

**Daylight Study** is the companion: the same room at midday. Warm paper ground,
ink-black type, the lamp darkened to a burnt amber. Not "dark mode inverted" —
a second designed palette.

### Rules the system enforces

1. **One display-size element per screen.** Everything else steps down hard.
2. **Three planes, not N cards.** `sunken` (the room) → `surface` (the desk) →
   `raised` (the lamp). Hover/focus promotes a plane; it does not add a border colour.
3. **Colour carries meaning, never decoration.** Warm is semantic — `primary`
   is the call to action, `urgent` is time pressure. Cool is identity — the
   class ramp. Everything else is neutral.
4. **Rhythm is uneven on purpose.** Section gaps are large, intra-panel gaps small.
   Uniform padding everywhere is the template smell we are removing.
5. **Motion only on state change, layout shift, or feedback.** Always compositor
   friendly (`transform`, `opacity`). Always collapsible to zero.

---

## 2. Colour

Semantic names only. Components ask for `--color-primary` or `--color-urgent`;
they never name a hue.

**The governing rule: warm is semantic, cool is identity.** Actions and urgency
own the warm end of the wheel; the class-identity ramp owns the cool end. A
colour on screen answers *"is this something to do?"* before you have read a
word of it — and a class can never be mistaken for a deadline.

### Roles

| Role | Meaning | Where it appears |
|---|---|---|
| `primary` | **Call to action.** The lamp. | Primary buttons, today's date, selected destination, the accent half of a display line |
| `primaryDeep` | Pressed primary | Primary button press state |
| `primaryWash` | Primary at low alpha | Tinted fill behind primary content |
| `onPrimary` | Drawn *on* a solid primary fill | Primary button label |
| `secondary` | Supporting emphasis | Secondary buttons, links, unfiled/neutral identity |
| `secondaryWash` | Secondary at low alpha | Secondary button fill |
| `urgent` | **Time pressure only** | Reminders, overdue, imminent countdowns, destructive actions, error messages |

At most one solid `primary` per surface. If two things on screen are both the
call to action, neither is.

Washes are expressed in CSS with `color-mix(in srgb, var(--color-primary) 12%, transparent)`
so the alpha lives in one place.

### Nocturne (dark) — a cool ink room lit by one warm lamp

| Token | Value | Use |
|---|---|---|
| `canvas` | `#08090D` | The room. App backdrop. |
| `surfaceSunken` | `#0B0D12` | Inset wells: calendar cells, inputs. |
| `surface` | `#101319` | The desk. Default panel ground. |
| `surfaceRaised` | `#171B23` | The lamp. Hover / focus / active panel. |
| `surfaceSidebar` | `#0A0C11` | Nav column. |
| `hairline` | `#FFFFFF` @ 8% | Default 1px separation. |
| `hairlineStrong` | `#FFFFFF` @ 15% | Emphasised separation. |
| `ink` | `#F2F1EC` | Primary text. |
| `inkSecondary` | `#B2BAC8` | Supporting text. |
| `inkFaint` | `#98A2B3` | Metadata. Clears 4.5:1 on all three planes. |
| `primary` | `#F7BC63` | Lamp amber. |
| `primaryDeep` | `#C48C36` | |
| `primaryWash` | primary @ 12% | |
| `onPrimary` | `#1A1206` | |
| `secondary` | `#A79AFF` | Periwinkle. |
| `secondaryWash` | secondary @ 12% | |
| `urgent` | `#FF8098` | Rose. |
| `urgentWash` | urgent @ 12% | |

### Daylight (light) — the same room at midday

Not "dark mode inverted" — a second designed palette. The lamp darkens to a
burnt amber so it still reads as *the* action colour against paper.

| Token | Value | Use |
|---|---|---|
| `canvas` | `#F4F1E9` | Warm paper. |
| `surfaceSunken` | `#EBE7DC` | |
| `surface` | `#FCFAF5` | |
| `surfaceRaised` | `#FFFFFF` | |
| `surfaceSidebar` | `#F0ECE1` | |
| `hairline` | `#12141A` @ 10% | |
| `hairlineStrong` | `#12141A` @ 18% | |
| `ink` | `#12141A` | |
| `inkSecondary` | `#4B5261` | |
| `inkFaint` | `#5C6472` | Clears 4.5:1 on all three planes. |
| `primary` | `#8A4F00` | Burnt amber. |
| `primaryDeep` | `#5F3700` | |
| `primaryWash` | primary @ 10% | |
| `onPrimary` | `#FFFFFF` | |
| `secondary` | `#473ABA` | Indigo. |
| `secondaryWash` | secondary @ 10% | |
| `urgent` | `#AD2340` | |
| `urgentWash` | urgent @ 10% | |

### Class identity ramp

Six cool stops, hues spread so no class is mistakable for another — and none of
them warm, so identity never reads as urgency.

| # | Nocturne | Daylight | Hue |
|---|---|---|---|
| 0 | `#7FB8FF` | `#1A5C9E` | sky |
| 1 | `#5FD3C8` | `#116B62` | teal |
| 2 | `#8FDC7B` | `#3A6F21` | green |
| 3 | `#D8CB6B` | `#6E5F0D` | wheat |
| 4 | `#E092D8` | `#923585` | orchid |
| 5 | `#B9A0F0` | `#54419E` | lavender |

Defined as `--color-identity-0` … `--color-identity-5`. Stops are assigned by an
item's **position in its list**, via a TypeScript `identityAt(index)` helper —
not by hashing its id. Hashing collided readily: five seeded classes drew only
three distinct stops, which defeats the ramp's one job. Anything not tied to a
class — a personal event, an unfiled note — takes `secondary` rather than
borrowing a class colour and implying a link.

### Contrast floor

Every text tier meets **4.5:1** against all three surface planes, in both
themes; so do `primary`, `secondary` and `urgent` as text, and `onPrimary` on a
solid primary fill. This must be verified by an automated contrast test over the
token values, not by eye — `inkFaint` failed this at 3.3:1 in an earlier build
and nothing caught it. (The frontend has no test runner yet; adding one is part
of implementing these tokens.)

---

## 3. Typography

**Two families, and only two.**
(Code blocks inside *previewed user files* use the system monospace stack,
`--font-code`; that is document content, not UI type.)

| Role | Family | Size / line | Weight | Notes |
|---|---|---|---|---|
| `display` | Fraunces | 56 / 1.02 | 600 | **Once per screen.** Tracking −1.0. |
| `displayAccent` | Fraunces | 56 / 1.02 | 600 italic | The `primary`-coloured half of a display line. |
| `title` | Work Sans | 22 / 1.25 | 600 | Panel headings. |
| `heading` | Work Sans | 17 / 1.30 | 600 | Sub-headings, list group titles. |
| `body` | Work Sans | 14.5 / 1.55 | 400 | |
| `bodyStrong` | Work Sans | 14.5 / 1.55 | 500 | |
| `small` | Work Sans | 12.5 / 1.45 | 400 | |
| `micro` | Work Sans | 11.5 | 600 | Eyebrows. Tracking 1.6, uppercase. |
| `numeric` | Work Sans | — | 500 | Tabular figures (`font-variant-numeric: tabular-nums`), so counters never jitter. |

Sizes are logical pixels; tracking values are px (`letter-spacing: -1px`,
`1.6px`). Each role is a set of properties, e.g. `--text-body-size`,
`--text-body-line`, `--text-body-weight`.

Fraunces sets the voice and appears exactly once per screen; Work Sans does
every other job, including figures and eyebrows. A third, monospaced family
previously carried labels and numbers — Work Sans' tabular numerals and wide
label tracking cover both, without adding another voice to the page.

The ramp's job is **scale contrast**: 56 → 22 → 14.5 is a real hierarchy. The
old 34 → 24 → 14 was not.

**Both families are bundled as assets** (`frontend/src/assets/fonts/` as
`woff2`, OFL texts alongside, loaded with `@font-face` and
`font-display: swap`), so typography never depends on a network fetch — a
runtime font CDN meant the type system silently degraded offline and on first
launch. Only the weights the ramp actually uses ship: Fraunces 600 + 600 italic,
Work Sans 400/500/600.

---

## 4. Space & shape

4-based scale. Applied unevenly — that is the point.

| Token | px | Typical use |
|---|---|---|
| `xxs` | 4 | Icon-to-label, marker gaps |
| `xs` | 8 | Tight stacks |
| `sm` | 12 | Control padding |
| `md` | 16 | Between sibling panels |
| `lg` | 24 | Inside panels |
| `xl` | 32 | Page gutter |
| `xxl` | 48 | Between page sections |
| `huge` | 72 | Above the hero only |

Radii — four values, not seven:

| Token | px | Use |
|---|---|---|
| `rXs` | 6 | Chips, markers, day cells |
| `rSm` | 10 | Buttons, inputs, nav items |
| `rMd` | 16 | Panels |
| `rLg` | 24 | Hero surfaces |

---

## 5. Motion

| Token | ms | Use |
|---|---|---|
| `instant` | 90 | Press feedback |
| `quick` | 160 | Hover, colour, small fades |
| `normal` | 240 | Panel transitions, route crossfade |
| `slow` | 420 | Entrance choreography |

| Curve | Flutter original | CSS |
|---|---|---|
| `standard` | `Curves.easeOutCubic` | `cubic-bezier(0.215, 0.61, 0.355, 1)` |
| `enter` | `Curves.easeOutExpo` | `cubic-bezier(0.19, 1, 0.22, 1)` |
| `exit` | `Curves.easeInCubic` | `cubic-bezier(0.55, 0.055, 0.675, 0.19)` |
| `emphasized` | `Cubic(0.2, 0.0, 0.0, 1.0)` | `cubic-bezier(0.2, 0, 0, 1)` |

**Press** is a spring (stiffness 520, damping 28) driving scale 0.985 → 1.0, so
release feels sprung rather than eased. CSS has no spring timing function, so
the spring is sampled into a `linear(…)` easing token (`--ease-press`) over its
settle time (`--motion-press`, 360 ms) and applied to `transform: scale()` only;
the press itself snaps in over `instant`. Both WebKit (macOS) and WebView2
(Windows) support `linear()`.

**Entrance choreography:** on view enter, regions stagger in at a 40 ms cadence,
each 12 px rise + fade over `slow`. One orchestration per view (a single parent
sets `--stagger-index` on its regions and drives `animation-delay`) — never an
independent transition per component.

**Reduced motion:** under `@media (prefers-reduced-motion: reduce)`, every
`--motion-*` duration token is set to `0ms`. Every animated rule reads its
duration from a token, and Svelte transitions take their `duration` from the
same values, so this is honoured everywhere by construction.

---

## 6. Elevation

There are no drop shadows on the dark theme — depth comes from plane + hairline.

| Plane | Dark | Light |
|---|---|---|
| Room | `canvas` + glows + dot grid | `canvas`, no glows |
| Desk | `surface` + `hairline` | `surface` + `hairline` |
| Lamp | `surfaceRaised` + `hairlineStrong` | `surfaceRaised` + soft shadow |

Hover promotes desk → lamp over `quick`. Focus does the same *and* draws a 2px
`primary` ring (`:focus-visible`) — so keyboard users get the state that
hover-only elements denied them.

---

## 7. Layout

Desktop-first. Breakpoints (CSS `@media (min-width: …)` on the webview width):

| Range | Sidebar | Content | Work area |
|---|---|---|---|
| ≥ 1440 | 280 expanded | max 1240 | calendar + rail side by side |
| 1180–1439 | 260 expanded | fluid | rail narrows to 320 |
| 1024–1179 | 72 icon rail | fluid | rail stacks under calendar |
| < 1024 | 72 icon rail | fluid | single column (graceful, not designed) |

The window's minimum size is 1000 × 640 (`main.go`), so the `< 1024` range is
reachable only at the narrowest window widths.

---

## 8. Composition rules

- **Metadata is not content.** Counts (`1 upcoming · 3 classes · 0 files`) render as
  one inline meta line under the greeting, not as three equal-weight tiles. The
  feature slot next to the hero holds the *actual next event*, rendered large.
- **Lists over grids** where items are heterogeneous. A 2-column wrap of identical
  rows is the template smell.
- **Empty states are designed**, never a bare sentence: eyebrow, one line of
  Fraunces, one action.
- **Every interactive surface** has hover, focus, pressed, and disabled states, all
  from a shared `PressableSurface.svelte` component. No `on:click={() => {}}`
  placeholders ship.
