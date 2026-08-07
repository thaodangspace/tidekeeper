# Tidekeepers — Frontend Web Specification

**Document type:** Product and engineering specification  
**Target:** MVP web application  
**Framework:** SvelteKit with TypeScript  
**Source gameplay document:** `tidekeepers-gameplay-design.md`  
**Status:** Draft ready for implementation  

---

## 1. Purpose

This document specifies the web frontend for **Tidekeepers**, a daily asynchronous strategy roguelite. The application lets a player:

1. Review yesterday's Tide result.
2. Understand why the Fleet won or lost.
3. Read today's Signals and rules.
4. Recruit, upgrade and arrange Keepers.
5. Select a one-day Strategy.
6. Lock the Fleet before the daily deadline.
7. Return after settlement to claim a reward and continue the Voyage.

The frontend is not authoritative for gameplay outcomes. It renders server state, collects player intent and provides immediate but reversible UI feedback. The backend remains the source of truth for inventory, Capital, snapshots, deadlines, market data, scores and rewards.

---

## 2. Product assumptions

### 2.1. MVP assumptions

- One active Voyage per player.
- One Voyage lasts seven Tide days.
- A Fleet starts with three slots and can grow to five slots.
- The daily cycle uses one server timezone and one global lock deadline.
- There is no real-time trading.
- There is no PvP in MVP.
- There is no AI dependency.
- Market-linked values are displayed as game metrics, not as investment advice.
- Keepers use fictional names and visuals.
- The player spends 3–5 minutes per daily session.

### 2.2. Canonical daily states

The frontend must use the backend-provided state without inferring it from the local clock alone.

```text
PREPARATION
→ LOCKED
→ SETTLING
→ RESULT_READY
→ REWARD_PENDING
→ PREPARATION for next day
```

Terminal Voyage states:

```text
COMPLETED
FAILED
ABANDONED
```

### 2.3. Internal and display vocabulary

| Internal/domain term | Player-facing label |
|---|---|
| Voyage | Voyage |
| Daily round | Tide |
| Lineup | Fleet |
| Capital | Supplies |
| Fund Health | Hull |
| Market benchmark | World Tide |
| Drawdown | Depth |
| Recovery | Resurface |
| Volatility | Turbulence |
| Asset basket | Current |

The API and TypeScript models may retain domain-oriented names such as `capital`, `fundHealth` and `lineup`; presentation components map them to the fantasy vocabulary.

---

## 3. Scope

### 3.1. In scope

- Authentication and session restoration.
- First-time onboarding.
- Voyage creation and continuation.
- Daily home/dashboard.
- Result reveal and score breakdown.
- Shop browsing and Keeper recruitment.
- Fleet editing.
- Keeper upgrades.
- Strategy selection.
- Signal and modifier review.
- Fleet lock confirmation.
- Locked and settlement states.
- Reward selection.
- Voyage progress and final summary.
- Settings, accessibility and legal disclaimers.
- Client analytics and error reporting.

### 3.2. Out of scope

- Real-time multiplayer.
- PvP matchmaking.
- Guilds and chat.
- Candlestick trading interface.
- Real-money purchases.
- Blockchain wallet connection.
- User-created markets or assets.
- Admin authoring UI.
- Rich combat canvas or Phaser integration.
- Push notifications in the first release.

---

## 4. Technical baseline

### 4.1. Core stack

- **SvelteKit** for routing, server rendering, form actions and deployment adapters.
- **Svelte with TypeScript** for UI components.
- **Vite** through SvelteKit for development and bundling.
- **Generated OpenAPI client** for typed backend requests.
- **Zod or equivalent schema validation** for client-side boundary checks when generated schemas are insufficient.
- **Vitest** for unit and component tests.
- **Playwright** for end-to-end tests.
- **Storybook for Svelte** or an equivalent isolated component environment, recommended but not release-blocking.

### 4.2. Rendering model

Use server rendering for:

- Session bootstrap.
- Initial daily state.
- Result pages reachable by URL.
- Public landing, legal and help pages.

Use client-side interaction for:

- Fleet drag-and-drop.
- Shop actions.
- Strategy selection.
- Reward choice.
- Expandable breakdowns.
- Result animation.

The app must remain usable without relying on long-lived browser memory. A refresh should reconstruct the current view from the backend.

### 4.3. Authentication transport

Preferred model:

- Backend creates an opaque session.
- Session token is stored in a `Secure`, `HttpOnly`, `SameSite=Lax` cookie.
- SvelteKit server requests forward the cookie to the API.
- Browser code never reads an access token.
- Mutating requests include CSRF protection where required by the backend design.

---

## 5. Frontend architecture

```text
Browser
  ↓
SvelteKit routes and layouts
  ├── server load functions
  ├── form actions / API proxy where appropriate
  ├── client interaction state
  └── presentational components
  ↓
Tidekeepers Go API
```

### 5.1. Architectural rules

1. **Server state is authoritative.** Never calculate final Capital, Hull, score, reward eligibility or lock status in the browser.
2. **Local state is ephemeral.** Local state may represent an uncommitted Fleet draft, selected tab, expanded details or animation progress.
3. **Mutations return the new canonical state.** After every successful mutation, replace affected local data with the response.
4. **Deadline enforcement is server-side.** The client countdown is informative only.
5. **No hidden game formulas.** The frontend displays breakdowns returned by the backend instead of recreating settlement math.
6. **Graceful reconnection.** Any failed mutation must be retryable without duplicating purchases or rewards.

### 5.2. State categories

#### Server state

- Current player.
- Active Voyage.
- Current Tide.
- Fleet snapshot or Fleet draft.
- Keeper inventory.
- Shop offers.
- Signals.
- Daily modifier and objective.
- Available Strategies.
- Rewards.
- Result breakdown.

#### Local UI state

- Selected Keeper.
- Drag source and target.
- Open modal or drawer.
- Active result tab.
- Animation already viewed.
- Unsaved Fleet ordering before autosave completes.
- Reduced-motion preference.

#### URL state

Use URL parameters only for shareable or restorable views:

- Result day number.
- Codex entry.
- Voyage summary tab.

Do not place sensitive state or game mutations in query parameters.

---

## 6. Recommended project structure

```text
src/
├── app.d.ts
├── hooks.server.ts
├── lib/
│   ├── api/
│   │   ├── client.server.ts
│   │   ├── client.browser.ts
│   │   ├── generated/
│   │   ├── errors.ts
│   │   └── mappers.ts
│   ├── auth/
│   ├── components/
│   │   ├── common/
│   │   ├── fleet/
│   │   ├── keeper/
│   │   ├── shop/
│   │   ├── signal/
│   │   ├── result/
│   │   ├── reward/
│   │   └── voyage/
│   ├── features/
│   │   ├── daily-home/
│   │   ├── preparation/
│   │   ├── settlement/
│   │   └── onboarding/
│   ├── stores/
│   ├── styles/
│   ├── types/
│   ├── utils/
│   └── validation/
└── routes/
    ├── +layout.server.ts
    ├── +layout.svelte
    ├── +page.svelte
    ├── login/
    ├── onboarding/
    ├── play/
    │   ├── +layout.server.ts
    │   ├── +page.server.ts
    │   ├── +page.svelte
    │   ├── prepare/
    │   ├── locked/
    │   ├── result/[day]/
    │   ├── reward/
    │   └── voyage/
    ├── codex/
    ├── settings/
    ├── help/
    └── legal/
```

Feature folders may own domain-specific components, action helpers and tests. Generic UI elements belong in `lib/components/common`.

---

## 7. Route specification

| Route | Purpose | Access |
|---|---|---|
| `/` | Public landing or redirect to active game | Public |
| `/login` | Login and account creation | Public |
| `/onboarding` | Explain daily loop and start first Voyage | Authenticated, first use |
| `/play` | Daily state router and primary dashboard | Authenticated |
| `/play/prepare` | Fleet, shop, Signals and Strategy | `PREPARATION` |
| `/play/locked` | Locked snapshot and countdown | `LOCKED` or `SETTLING` |
| `/play/result/[day]` | Result reveal and breakdown | Result available |
| `/play/reward` | Select one reward | `REWARD_PENDING` |
| `/play/voyage` | Voyage map, Hull, Relics and history | Active Voyage |
| `/codex` | Keeper, Relic, Strategy and rule glossary | Authenticated |
| `/settings` | Preferences, account and accessibility | Authenticated |
| `/help` | Rules and disclaimers | Public |
| `/legal/*` | Terms, privacy and game-data disclaimer | Public |

`/play` should redirect based on the backend state, not only on route guards:

```text
PREPARATION   → /play/prepare
LOCKED        → /play/locked
SETTLING      → /play/locked
RESULT_READY  → /play/result/{day}
REWARD_PENDING→ /play/reward
COMPLETED     → /play/voyage?view=summary
FAILED        → /play/voyage?view=summary
```

---

## 8. Shared frontend data model

The generated API types are canonical. The following types describe the minimum expected shape.

```ts
export type DailyPhase =
  | 'PREPARATION'
  | 'LOCKED'
  | 'SETTLING'
  | 'RESULT_READY'
  | 'REWARD_PENDING';

export type VoyageStatus =
  | 'ACTIVE'
  | 'COMPLETED'
  | 'FAILED'
  | 'ABANDONED';

export interface DailyContext {
  voyageId: string;
  dayNumber: number;
  phase: DailyPhase;
  lockAt: string;
  settleAfter: string;
  serverNow: string;
  modifier: DailyModifier;
  objective: DailyObjective;
  signals: Signal[];
  lineup: LineupView;
  inventory: KeeperInstance[];
  shop?: ShopView;
  strategies: StrategyView[];
  selectedStrategyId?: string;
  capital: number;
  fundHealth: number;
  maxFundHealth: number;
  pendingRewardCount: number;
  version: number;
}

export interface LineupView {
  maxSlots: number;
  slots: Array<{
    index: number;
    keeperInstanceId?: string;
  }>;
  synergies: SynergyView[];
  warnings: LineupWarning[];
  lockedAt?: string;
}

export interface KeeperInstance {
  id: string;
  definitionId: string;
  name: string;
  sector: string;
  role: string;
  rarity: string;
  level: number;
  passiveSummary: string;
  upgradeOptions?: UpgradeOption[];
  artworkUrl?: string;
}
```

All timestamps are ISO 8601 strings in UTC. Formatting into the user's locale occurs only in presentation utilities.

---

## 9. Screen specifications

## 9.1. Authentication

### Required elements

- Tidekeepers logo and short promise.
- Email login or configured identity provider.
- Clear loading and error states.
- Links to terms, privacy and gameplay disclaimer.

### Acceptance criteria

- Existing sessions skip login.
- An expired session returns the player to login without losing server state.
- Errors never expose raw backend messages or stack traces.

---

## 9.2. Onboarding

Onboarding should be interactive and take less than three minutes.

### Steps

1. Explain `Build today. Face tomorrow.`
2. Explain Hull, Supplies and the seven-day Voyage.
3. Show three starter Keepers.
4. Let the player place starter Keepers in the Fleet.
5. Show one sample Signal and one Daily Modifier.
6. Create the Voyage and enter preparation.

### Requirements

- The player can review definitions through tooltips.
- The player cannot create duplicate active Voyages.
- Server creation is idempotent.
- Onboarding completion is stored server-side.

---

## 9.3. Daily home

The daily home is the highest-priority screen and should answer four questions above the fold:

1. What happened?
2. What must I do now?
3. How much time remains?
4. What is at risk?

### Header

- Current Voyage and day number.
- Hull meter.
- Supplies count.
- Countdown based on `lockAt` and `serverNow`.
- Current phase badge.

### Primary card by phase

#### `RESULT_READY`

- Win, survival or loss headline.
- Hull change.
- Supplies earned.
- Primary CTA: `View result`.

#### `REWARD_PENDING`

- Reward reminder.
- Primary CTA: `Choose reward`.

#### `PREPARATION`

- Daily Modifier.
- Objective.
- Signal summary.
- Fleet readiness.
- Primary CTA: `Prepare Fleet`.

#### `LOCKED`

- Locked Fleet preview.
- Settlement countdown.
- Primary CTA: `Review locked Fleet`.

#### `SETTLING`

- Processing state.
- Last successful refresh.
- Automatic polling status.

---

## 9.4. Result reveal

Result reveal is the core retention moment.

### Reveal sequence

1. Display Tide name and regime.
2. Animate World Tide versus Fleet performance.
3. Reveal major Keeper triggers.
4. Reveal final score and outcome.
5. Show Hull and Supplies changes.
6. Enable detailed breakdown.

Animation must be skippable and respect reduced-motion preferences.

### Breakdown tabs

#### Summary

- Outcome.
- Daily score.
- Threshold.
- Hull change.
- Supplies gained.

#### Timeline

- World Tide line.
- Fleet line.
- Max Depth marker.
- Resurface marker.
- Skill trigger markers.

This is a game visualization, not a trading chart. Avoid candlestick visuals.

#### Keepers

For every Keeper:

- Raw Current result.
- Global relative result.
- Sector relative result.
- Risk-adjusted result.
- Skill contribution.
- Penalty.

#### Rules

- Modifier contribution.
- Objective completion.
- Synergy contribution.
- Relic contribution.
- Strategy contribution.

### Required explanations

Every numerical score must have:

- A short human-readable explanation.
- An optional expanded formula input list.
- No implication of investment performance.

---

## 9.5. Preparation workspace

Desktop layout:

```text
┌──────────────────────────────────────────────────────────┐
│ Header: Hull | Supplies | Deadline | Lock Fleet          │
├───────────────┬────────────────────────┬─────────────────┤
│ Signals       │ Fleet                  │ Daily rules     │
│ and Objective │ 3–5 Keeper slots       │ Modifier        │
│               │ Synergies              │ Strategy        │
├───────────────┴────────────────────────┴─────────────────┤
│ Shop / Inventory tabs                                   │
└──────────────────────────────────────────────────────────┘
```

Mobile layout uses stacked sections with a sticky summary and sticky lock CTA.

### Fleet interaction

Support:

- Drag-and-drop on pointer devices.
- Tap-to-select then tap-slot on touch and keyboard.
- Move Keeper between slots.
- Remove Keeper to bench.
- Show active and near-active Synergies.
- Show warnings before lock.

A Fleet update should autosave after a short debounce or use explicit server mutation per move. The backend must receive an expected version for optimistic concurrency.

### Unsaved and conflict behavior

- Show `Saving…`, `Saved` or `Could not save` near the Fleet title.
- On version conflict, fetch the latest Fleet and explain that it changed elsewhere.
- Never silently discard a server-confirmed state.
- Before route leave, retry a failed save or show a blocking warning.

---

## 9.6. Shop

### Shop card content

- Keeper portrait.
- Name.
- Rarity.
- Sector and Role.
- Cost.
- Concise passive.
- Whether recruitment activates or advances a Synergy.

### Actions

- Recruit.
- Inspect.
- Reroll.
- Upgrade, when the design exposes upgrades in the shop.

### Rules

- Disable actions while a request is pending.
- Use an idempotency key for recruit and reroll mutations.
- Do not permanently decrement Supplies until the server confirms.
- On `INSUFFICIENT_CAPITAL`, restore the canonical balance from the response.
- On deadline expiry, move the player into the current server phase.

---

## 9.7. Keeper detail

Displayed as a responsive drawer or modal.

### Content

- Artwork and fantasy description.
- Sector and Role.
- Passive text.
- Current upgrade.
- Available branch upgrades.
- Expected behavior in each regime.
- Current Synergy relationships.
- Data disclaimer in a collapsed help area.

No direct real-world asset names should appear in the main gameplay view. A transparency page may describe the methodology at a basket level.

---

## 9.8. Signals and daily rules

### Signal card

- Name.
- Direction or category.
- Strength band, not false precision.
- Plain-language interpretation.
- Source time range.
- Confidence label only if produced by a deterministic rule.

### Modifier card

- Rule title.
- Benefited Roles or Sectors.
- Penalized Roles or Sectors.
- Example.

### Objective card

- Objective statement.
- Completion threshold.
- Reward or score contribution.

Do not display a deterministic Signal as a promise about future values.

---

## 9.9. Strategy selection

- Display all available one-day Strategies.
- Only one may be active unless backend rules say otherwise.
- Show upside and downside with equal visual weight.
- Selecting a Strategy updates the preparation state through the API.
- Changing Strategy is allowed until lock.
- Lock confirmation includes the selected Strategy.

---

## 9.10. Lock confirmation

Locking is a high-consequence action and requires confirmation.

### Modal content

- Exact server deadline.
- Fleet slots.
- Active Synergies.
- Selected Strategy.
- Unresolved warnings.
- Explicit statement that changes are not allowed after lock.

### Required mutation behavior

- Send a unique idempotency key.
- Include the expected daily-context version.
- Disable duplicate submissions.
- On success, redirect to `/play/locked` with canonical snapshot data.
- On deadline conflict, show the server phase and redirect accordingly.

---

## 9.11. Locked and settling

### `LOCKED`

- Locked snapshot checksum or lock timestamp.
- Fleet and Strategy.
- Daily Modifier and Objective.
- Countdown until settlement window begins.
- No editing controls.

### `SETTLING`

- Calm processing animation.
- `Checking the Tide…` status.
- Poll every 20–30 seconds with exponential backoff after failures.
- Stop polling when the tab is hidden for a prolonged period, then refresh on focus.
- Provide manual refresh after a reasonable delay.

WebSocket infrastructure is unnecessary for MVP. Polling is sufficient because settlement occurs once per day.

---

## 9.12. Reward selection

### Reward types

- Recruit a Keeper.
- Upgrade a Keeper.
- Gain a Relic.
- Gain Supplies.
- Restore Hull.
- Unlock a Fleet slot.
- Preview tomorrow's Modifier.

### Requirements

- Display exactly the server-generated choices.
- Reward claim is idempotent and irreversible.
- Explain immediate impact before confirmation.
- On success, route to the next preparation or Voyage summary.
- If the reward was already claimed from another device, show the claimed reward and continue.

---

## 9.13. Voyage view

### Active Voyage

- Day nodes and state.
- Current Hull.
- Relics.
- Fleet history.
- Previous daily outcomes.
- Upcoming known Bosses.

### Final summary

- Completed or failed state.
- Voyage score.
- Days survived.
- Strongest Synergy.
- Most impactful Keeper.
- Largest Depth.
- Best recovery.
- Restart CTA.

---

## 10. API integration

### 10.1. Base conventions

- Base path: `/api/v1`.
- JSON request and response bodies.
- ISO 8601 UTC timestamps.
- Error envelope with stable machine codes.
- `Idempotency-Key` header on consequential mutations.
- `If-Match` or explicit `version` field on mutable daily state.
- Correlation ID returned in `X-Request-ID`.

### 10.2. Minimum endpoint usage

| Frontend action | Endpoint |
|---|---|
| Bootstrap player | `GET /me` |
| Load active daily state | `GET /voyages/current/daily-context` |
| Create Voyage | `POST /voyages` |
| Update Fleet | `PUT /voyages/{id}/days/{day}/lineup` |
| Recruit Keeper | `POST /voyages/{id}/days/{day}/shop/recruit` |
| Reroll shop | `POST /voyages/{id}/days/{day}/shop/reroll` |
| Select Strategy | `PUT /voyages/{id}/days/{day}/strategy` |
| Upgrade Keeper | `POST /voyages/{id}/keepers/{instanceId}/upgrade` |
| Lock Fleet | `POST /voyages/{id}/days/{day}/lock` |
| Load result | `GET /voyages/{id}/days/{day}/result` |
| Claim reward | `POST /voyages/{id}/days/{day}/reward-claims` |
| Load Voyage | `GET /voyages/{id}` |
| Abandon Voyage | `POST /voyages/{id}/abandon` |

### 10.3. Error codes the UI must handle

```text
AUTH_REQUIRED
SESSION_EXPIRED
VOYAGE_NOT_FOUND
NO_ACTIVE_VOYAGE
INVALID_PHASE
DEADLINE_PASSED
VERSION_CONFLICT
LINEUP_INVALID
KEEPER_NOT_OWNED
KEEPER_ALREADY_ASSIGNED
INSUFFICIENT_CAPITAL
SHOP_OFFER_EXPIRED
STRATEGY_UNAVAILABLE
ALREADY_LOCKED
RESULT_NOT_READY
REWARD_ALREADY_CLAIMED
PROVIDER_DATA_PENDING
RATE_LIMITED
INTERNAL_ERROR
```

The API client maps codes into user-facing messages. Avoid branching on free-form text.

---

## 11. Data fetching and cache policy

### Initial load

Use SvelteKit server `load` functions to fetch:

- Session.
- Active Voyage.
- Daily context.

### Browser refresh policy

- Preparation state: refresh on window focus if older than 60 seconds.
- Locked state: refresh on focus and before deadline display reaches zero.
- Settling state: poll every 20–30 seconds.
- Result and historical pages: cache until game rule or result version changes.
- Codex: long cache, versioned by content release.

### Optimistic behavior

Allowed:

- Reordering Fleet slots.
- Selecting tabs.
- Expanding details.
- Highlighting a tentative Strategy.

Not allowed before server confirmation:

- Spending Supplies.
- Adding a Keeper to inventory.
- Increasing Hull.
- Claiming a reward.
- Marking Fleet as locked.

---

## 12. Time and deadline handling

- Backend returns `serverNow`, `lockAt` and `settleAfter`.
- Client computes an offset between local time and `serverNow`.
- Countdown uses the offset-adjusted server time.
- Re-sync time on window focus and every five minutes during preparation.
- When countdown reaches zero, disable mutations and immediately refresh server state.
- Display the server timezone in the lock confirmation.
- Use exact dates and times in tooltips; use relative countdowns in the primary UI.

---

## 13. Forms and validation

- Validate required fields in the browser for responsiveness.
- Treat backend validation as final.
- Keep validation messages next to the responsible control.
- Fleet validation warnings should distinguish:
  - Blocking errors.
  - Strategic warnings.
  - Informational suggestions.

Example:

```text
Blocking: Slot 2 references a Keeper no longer owned.
Warning: No Warden; projected Depth control is low.
Info: Adding one Crest Keeper activates Crest II.
```

---

## 14. Visual system

### 14.1. Direction

- Maritime fantasy with readable modern UI.
- Avoid visual resemblance to a financial terminal.
- Use waves, currents, compasses, hulls and constellations as metaphors.
- Keep numerical information prominent and legible.

### 14.2. Design tokens

Define tokens for:

- Surface levels.
- Text hierarchy.
- Positive, negative, neutral and warning semantics.
- Sector identities.
- Role identities.
- Rarity.
- Spacing.
- Radius.
- Elevation.
- Motion duration.

Do not rely on color alone for positive or negative outcomes. Pair color with icons, labels and shape.

### 14.3. Component inventory

- App shell.
- Status header.
- Hull meter.
- Supplies badge.
- Deadline timer.
- Phase badge.
- Keeper card.
- Keeper portrait.
- Fleet slot.
- Synergy chip.
- Signal card.
- Modifier card.
- Objective card.
- Strategy card.
- Relic chip.
- Score contribution row.
- Timeline chart.
- Reward card.
- Confirmation modal.
- Toast and inline alert.
- Empty, loading and error states.

---

## 15. Responsive behavior

### Desktop

- Three-column preparation workspace.
- Persistent Fleet and daily rules.
- Shop visible below or in a side panel.

### Tablet

- Two-column layout.
- Signals and rules may share tabs.
- Fleet remains fully visible.

### Mobile

- Single-column flow.
- Sticky compact status bar.
- Sticky primary CTA.
- Horizontal Fleet slot rail or compact grid.
- Bottom sheets for Keeper details and Strategy selection.
- No interaction depends exclusively on hover.

Minimum supported viewport should be defined by product analytics; default implementation target is 360 CSS pixels wide.

---

## 16. Accessibility

Target WCAG 2.2 AA behavior.

- Full keyboard navigation.
- Visible focus indicators.
- Semantic buttons and headings.
- Drag-and-drop has keyboard and touch alternatives.
- Live region for save, lock and reward status.
- Reduced-motion mode.
- Charts have text summaries.
- Color contrast meets accessibility targets.
- Tooltips are reachable by keyboard and touch.
- Countdown updates do not announce every second.
- Result animations are skippable.

---

## 17. Localization

- All player-facing strings use a translation layer from the start.
- English and Vietnamese should be supported by structure, even if only one ships first.
- Do not concatenate translated sentence fragments.
- Dates use the player's locale while preserving the named server timezone where relevant.
- Number formatting uses locale-aware utilities.
- Fantasy terms have glossary keys rather than hard-coded strings.

---

## 18. Security and privacy

- Never place session tokens in `localStorage`.
- Escape all server-provided text by default.
- Restrict artwork and media hosts through Content Security Policy.
- Do not expose provider API keys.
- Do not log full session cookies or personal data.
- Use CSRF mitigation for cookie-authenticated mutations.
- Require confirmation for Voyage abandonment.
- Legal pages must state that the game does not provide investment advice and does not involve ownership of real assets.

---

## 19. Performance requirements

Targets for a typical authenticated route on a mid-range mobile device:

- Initial HTML should contain meaningful daily state.
- Avoid loading all artwork at full resolution.
- Lazy-load Codex and historical result assets.
- Result timeline should use lightweight SVG or Canvas only when necessary.
- Preparation interactions should respond within 100 ms locally.
- A successful mutation should visibly acknowledge within 300 ms when network allows.
- JavaScript for public landing and authenticated gameplay should be split.

Use responsive images, route-level code splitting and asset preloading only for the next likely view.

---

## 20. Offline and degraded behavior

MVP is online-first.

### Temporary network loss

- Keep the last confirmed state visible.
- Clearly label it as potentially stale.
- Queue only safe, reversible UI actions locally.
- Do not queue purchases, locks or reward claims for silent replay.
- Provide explicit retry.

### Backend unavailable

- Show a service status message.
- Preserve the current route.
- Include the request ID when available.

### Data settlement pending

- Distinguish provider-data delay from general failure.
- Show `Result pending; your locked Fleet is safe.`

---

## 21. Analytics events

Minimum events:

```text
session_started
onboarding_started
onboarding_completed
voyage_created
result_viewed
result_breakdown_opened
keeper_inspected
keeper_recruited
shop_rerolled
lineup_changed
strategy_selected
fleet_lock_started
fleet_locked
fleet_lock_failed
reward_viewed
reward_claimed
voyage_completed
voyage_failed
voyage_abandoned
```

Event properties may include:

- Voyage day.
- Phase.
- Keeper definition ID.
- Sector and Role.
- Synergy count.
- Remaining Hull band.
- Time before deadline band.

Do not send raw session identifiers, provider payloads or unnecessary personal data.

---

## 22. Error reporting

- Capture uncaught browser errors.
- Capture failed route loads.
- Attach `X-Request-ID` when available.
- Redact cookies and personal information.
- Group errors by stable API error code.
- Provide a user-visible recovery action for every blocking error.

---

## 23. Testing strategy

### Unit tests

- Time offset and countdown utilities.
- API error mapping.
- Score formatting.
- Fleet warning rendering.
- Domain-to-display terminology mapping.

### Component tests

- Keeper card states.
- Fleet keyboard interaction.
- Lock confirmation.
- Reward card behavior.
- Result breakdown accordions.
- Reduced-motion result reveal.

### Integration tests

- Server load with authenticated session.
- Version conflict recovery.
- Deadline transition.
- Mutation error handling.
- Polling from `SETTLING` to `RESULT_READY`.

### End-to-end critical paths

1. New player creates a Voyage.
2. Player recruits a Keeper and edits the Fleet.
3. Player selects Strategy and locks Fleet.
4. Player returns to a ready result.
5. Player understands breakdown and claims reward.
6. Player completes the seven-day Voyage.
7. Expired session is recovered safely.
8. Duplicate lock and reward submissions do not duplicate effects.
9. Mobile keyboard-only and touch flows remain usable.

---

## 24. Environment variables

```text
PUBLIC_APP_ENV
PUBLIC_API_BASE_URL
PUBLIC_ASSET_BASE_URL
PUBLIC_ANALYTICS_ENABLED
PUBLIC_SUPPORT_URL
INTERNAL_API_BASE_URL
SESSION_COOKIE_NAME
CSRF_COOKIE_NAME
ERROR_REPORTING_DSN
```

Secrets must not use the `PUBLIC_` prefix.

---

## 25. Deployment

- Build a SvelteKit application using an adapter suited to the chosen runtime.
- Run at least one server-rendering instance for authenticated routes.
- Serve immutable assets through a CDN.
- Use health checks for the web process.
- Version frontend releases and expose the version in a diagnostics panel.
- Keep frontend and backend deployable independently while validating OpenAPI compatibility in CI.

---

## 26. CI quality gates

A merge to the release branch requires:

- Type checking.
- Linting and formatting.
- Unit tests.
- Component tests for changed critical components.
- Production build.
- OpenAPI client generation with no uncommitted diff.
- Accessibility checks on primary routes.
- Playwright smoke test against an ephemeral backend environment.

---

## 27. MVP acceptance criteria

The frontend MVP is complete when:

1. A new player can register, complete onboarding and start a Voyage.
2. The current daily phase always routes to the correct screen.
3. The player can inspect Signals, Modifier and Objective.
4. The player can recruit Keepers, manage the Fleet and select a Strategy.
5. The player cannot make server-accepted changes after lock.
6. Fleet locking is idempotent and has a clear confirmation.
7. The player can return later and view a result with an understandable breakdown.
8. The player can claim exactly one reward and continue.
9. The seven-day Voyage can complete or fail with a summary.
10. Refreshing or changing devices does not lose confirmed progress.
11. The primary flow is usable on desktop and mobile.
12. The primary flow is usable with keyboard and reduced motion.
13. Errors provide recovery and never expose sensitive implementation details.

---

## 28. Suggested delivery sequence

### Milestone 1 — Read-only shell

- Authentication.
- App shell.
- Daily-context rendering.
- Phase routing.
- Static Keeper and rule components.

### Milestone 2 — Preparation

- Fleet editing.
- Shop.
- Strategy selection.
- Save and version-conflict behavior.
- Lock confirmation.

### Milestone 3 — Result and rewards

- Result reveal.
- Timeline and breakdown.
- Reward selection.
- Voyage progression.

### Milestone 4 — Hardening

- Accessibility.
- Responsive polish.
- Analytics.
- Error reporting.
- End-to-end tests.
- Performance optimization.

---

## 29. Product decisions intentionally deferred

- Native mobile application.
- Push notification provider.
- Rich 2D battle animation.
- PvP comparison screen.
- Social sharing.
- Multiple simultaneous Voyages.
- User-selected server regions.
- Public market-methodology dashboard.
- Offline mutation queue.

These decisions should not block the MVP architecture.
