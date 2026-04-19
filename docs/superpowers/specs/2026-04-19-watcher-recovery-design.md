# Watcher Recovery Design

## Goal

Restore the revenue-critical watcher path inside `translation-app` so the system can:

- maintain a long-lived per-user browser identity on the server
- monitor Gengo RSS and realtime websocket feeds reliably and observably
- open matching job pages in the authoritative worker browser
- alert immediately and visibly on dangerous states such as captcha or suspicious login friction
- present a web UI that makes current state, recent actions, and failure reasons obvious to the operator

This recovery project intentionally prioritizes reliability, observability, and safe action execution over breadth. Features such as cancellation, email/site watchers, remote takeover, and multi-browser support are deferred until the core moneymaking loop is stable.

## Product Scope

### In scope for recovery

- one long-lived browser worker per user
- persistent browser profile per user, stored on the server
- initial worker seeding by importing an existing browser profile/state
- Google OAuth session continuity inside the worker browser
- RSS monitoring
- Gengo realtime websocket monitoring
- structured event timeline and worker runtime snapshots
- worker browser screenshot/preview visibility
- worker-side job opening flow
- worker-side accept click flow for matching jobs
- immediate hard-stop on captcha, suspicious-login, or unexpected safety-critical page states
- web dashboard inside `translation-app`
- per-user remote notifications to connected clients

### Explicitly out of scope for recovery

- remote browser takeover/control by the user
- Firefox-family browser backend
- email watcher parity
- website watcher parity
- cancellation parity
- advanced page evaluation logic
- distributed worker fleet
- multi-machine orchestration

## Key Constraints

### Browser identity and stealth

The worker must use a real browser process controlled through a DevTools-style remote debugging interface. It must preserve a stable, long-lived profile per user rather than using ephemeral sessions. This worker browser is the authoritative identity for job-opening and acceptance actions.

The backend must not depend on syncing a user’s local live browser into the flow during normal operation. The server-owned worker profile is the primary runtime identity. Initial seeding can come from an imported real profile, but day-to-day operation must remain server-owned.

### Safety

If the worker encounters Turnstile/captcha, suspicious-login flow, account-security flow, or an unexpected page state during action execution, it must:

1. stop further actions immediately
2. mark the worker as blocked
3. emit a critical alert event
4. capture a screenshot or equivalent artifact
5. surface the block clearly in the UI

No automatic recovery should be attempted in the first production version for these states.

### Deployment

The first production version should assume a single server hosting:

- the `translation-app` API and frontend
- the watcher runtime
- all browser worker processes
- persisted profile/state storage

However, the boundaries should be clean enough that browser workers can be split out later.

## Architecture Overview

`translation-app` becomes the single product boundary and source of truth. The system is composed of five main subsystems:

1. `Worker Supervisor`
2. `Feed Monitors`
3. `Browser Worker`
4. `Action Coordinator`
5. `Observability and Delivery Layer`

### 1. Worker Supervisor

The Worker Supervisor owns one logical watcher per user. It is responsible for:

- desired-vs-actual worker state
- startup and shutdown of subcomponents
- health aggregation
- restart policy for non-safety failures
- transitions between `seeded`, `starting`, `ready`, `busy`, `degraded`, `blocked`, and `stopped`

The supervisor should treat feeds, browser, and actions as distinct health domains rather than collapsing everything into one “running” boolean.

### 2. Feed Monitors

The feed layer contains:

- `RSSMonitor`
- `GengoWSMonitor`

These monitors are responsible only for discovering job opportunities and reporting feed health. They do not own browser behavior.

### 3. Browser Worker

Each user gets one long-lived Browser Worker. It owns:

- the persisted browser profile directory
- the browser process lifecycle
- the DevTools/CDP connection
- browser state introspection
- browser actions such as open job page, click accept, capture screenshot

The browser worker does not decide product policy. It executes commands and emits detailed state.

### 4. Action Coordinator

The Action Coordinator receives discovered jobs and serializes the action path:

- threshold/rule check
- queueing
- open job page in the worker browser
- safety checks
- accept click
- result classification

This layer ensures the worker never performs conflicting actions concurrently.

### 5. Observability and Delivery Layer

This layer persists runtime snapshots and events, pushes them over the existing per-user websocket channel, and powers the dashboard.

The operator UI should read from authoritative runtime state rather than inferring status client-side from ad hoc logs.

## Worker Model

## Long-lived per-user worker

Each user should have exactly one long-lived worker in the recovery design.

Each worker owns:

- `worker_id`
- `user_id`
- persisted profile path
- browser binary metadata
- current runtime state
- action queue
- recent artifacts
- recent events

This is a stable identity, not an on-demand spawned disposable session.

## Browser backend

The first Go-native implementation should target Chromium-family browsers only.

Reasons:

- CDP/DevTools support is mature and straightforward
- Google OAuth flow support is well understood in Chromium-family browsers
- the stealth model maps more directly to a DevTools-controlled Chromium browser
- it reduces surface area for the revenue-recovery release

Firefox-family support can be added later as a second backend, but it should not delay the first production recovery path.

## Worker lifecycle states

Each worker should expose a strongly typed state machine, at minimum:

- `Seeded`
- `Starting`
- `Attached`
- `Ready`
- `Busy`
- `Degraded`
- `Blocked`
- `Stopped`

### State meanings

- `Seeded`: profile exists but browser/login state not yet verified
- `Starting`: browser process launch in progress
- `Attached`: browser launched and DevTools attached, but readiness checks still running
- `Ready`: browser healthy, attached, and verified logged into Gengo
- `Busy`: worker is executing a job action
- `Degraded`: worker partially functioning but not healthy enough to be trusted fully
- `Blocked`: safety-critical issue detected, no further actions allowed
- `Stopped`: worker intentionally not running

## Initial seeding

For the recovery version, the operator can import an existing real browser profile/state into the server. That imported state becomes the persisted worker profile.

Later, a first-run server-hosted login flow can be added for ordinary users, but recovery does not depend on it.

## Data Model

The system should use `runtime snapshot + append-only events + artifacts` as its core observability model.

## Runtime snapshot

Persist a per-worker `WorkerRuntimeState` that answers: “what is happening now?”

Fields should include at least:

- `worker_id`
- `user_id`
- `overall_status`
- `feed_status`
- `browser_status`
- `action_status`
- `alert_status`
- `current_job_id`
- `current_action_step`
- `current_url`
- `current_title`
- `logged_in_state`
- `browser_process_alive`
- `devtools_connected`
- `last_rss_poll_started_at`
- `last_rss_poll_ok_at`
- `rss_consecutive_failures`
- `last_ws_connect_at`
- `last_ws_message_at`
- `last_ws_pong_at`
- `last_ws_close_code`
- `last_ws_close_reason`
- `ws_reconnect_count`
- `last_browser_heartbeat_at`
- `last_error`
- `last_critical_alert`
- `latest_screenshot_artifact_id`
- `updated_at`

The dashboard should primarily render from this snapshot.

## Events

Persist append-only `WorkerEvent` records that answer: “how did we get here?”

Fields:

- `id`
- `worker_id`
- `user_id`
- `timestamp`
- `level` (`info`, `warning`, `critical`)
- `source` (`rss`, `gengo_ws`, `browser`, `action`, `system`)
- `type`
- `job_id` nullable
- `message`
- `data` JSON payload

Example event types:

- `worker.started`
- `worker.ready`
- `rss.poll_started`
- `rss.poll_ok`
- `rss.poll_failed`
- `ws.connecting`
- `ws.authenticated`
- `ws.message_received`
- `ws.quiet`
- `job.detected`
- `job.matched`
- `browser.job_open_started`
- `browser.job_open_succeeded`
- `action.accept_started`
- `action.accept_succeeded`
- `action.accept_failed`
- `browser.captcha_detected`
- `browser.suspicious_login_detected`
- `worker.blocked`

## Artifacts

Persist references to selected artifacts for investigation and UI display:

- screenshots
- optionally page HTML snapshots for failures later

For recovery, storing screenshots is sufficient. Keep the latest screenshot plus key failure screenshots rather than attempting infinite retention.

## Feed Reliability Model

The feed layer must be explicit, supervised, and independently observable.

## RSS Monitor

Responsibilities:

- poll Gengo RSS on a configured cadence with jitter
- record start, success, and failure times
- parse candidate jobs
- deduplicate and emit structured `job.detected` events
- report failures and consecutive failure counts

Runtime state to expose:

- polling cadence
- last poll start
- last successful poll
- last error
- consecutive failures
- next scheduled poll

## Gengo WebSocket Monitor

Responsibilities:

- establish and authenticate a connection to Gengo’s realtime websocket
- track heartbeat and quiet-socket health
- emit structured realtime job detection events
- reconnect with backoff on ordinary failures
- expose detailed close and failure metadata

Runtime state to expose:

- connect/auth status
- connected since
- last message time
- last pong time
- reconnect count
- last close code and reason
- quiet-socket detection state

## Health domains

Do not expose only a single `watcher_status`. Expose separate domains:

- `Feeds`
- `Browser`
- `Action`
- `Alerts`

Each should be derivable from concrete runtime fields.

Examples:

- feeds healthy, browser healthy, action idle
- feeds degraded, browser healthy, action idle
- feeds healthy, browser blocked, alert critical
- feeds healthy, browser healthy, action failed

## Action Model

The revenue-critical action path is deliberately simple in the first release.

## Action path

For a matching job:

1. job arrives from RSS or Gengo websocket
2. job is normalized and deduplicated
3. threshold/rule check decides whether it should trigger action
4. Action Coordinator queues the job
5. Browser Worker opens the job page in the authoritative worker browser
6. worker waits for expected page readiness
7. worker checks for hard-stop indicators first
8. if safe, worker clicks the known accept control
9. worker classifies the result and emits structured events

The worker does not need to “evaluate” job value from the page. Gengo’s notification payload already contains the relevant value for this recovery path.

## Action serialization

A worker must process browser actions through a single serialized queue. This avoids:

- overlapping job opens
- racing accept clicks
- state corruption in the worker browser

## Result classification

Every action attempt should resolve into a narrow, explicit outcome set:

- `accepted`
- `already_gone`
- `blocked_captcha`
- `blocked_suspicious_login`
- `unexpected_page_state`
- `browser_failure`
- `timeout`

The operator UI should render these directly rather than inferring from logs.

## Hard-stop conditions

The following should immediately stop the worker’s action pipeline and mark it blocked:

- Turnstile or captcha markers detected
- suspicious-login prompts
- account security verification prompts
- redirect to login during an action flow
- expected accept control missing on a page that should contain it
- page state suggesting account risk or anti-bot escalation

No retry or session-refresh recovery should happen automatically in recovery mode.

## UI Design

The watcher UI should be a dedicated operations console inside `translation-app`, not merely a configuration form with logs.

The main question it must answer is:

`Am I protected right now, and if not, exactly what is broken?`

## Main layout

The primary operations view should contain six persistent areas.

### 1. Top status bar

This is the “at a glance” strip.

It should show:

- worker identity
- overall worker state
- `Feeds` status
- `Browser` status
- `Action` status
- `Alerts` status
- worker uptime
- last activity time
- profile health (`seeded`, `verified`, `blocked`, `relogin required`)

This bar should be readable in seconds.

### 2. Current action panel

This is the dominant operational panel when something is happening.

It should show the live action progression, for example:

- `Job detected`
- `Queued`
- `Opening page`
- `Page loaded`
- `Safety checks passed`
- `Accept clicked`
- `Accepted`

If the flow fails, the panel should freeze on the failing step and show the exact classified reason.

If nothing is happening, it should clearly show `Idle, monitoring feeds`.

### 3. Browser panel

This should show:

- latest screenshot or refreshed browser preview
- current URL
- current page title
- last screenshot time
- login/session state
- browser process state
- DevTools/CDP state
- any detected challenge flags

Even without remote takeover, this gives enough visibility to understand what the worker is actually doing.

### 4. Feed health panel

Separate cards for RSS and realtime websocket.

RSS card:

- last poll start
- last success
- consecutive failures
- next poll
- last error

Websocket card:

- connect state
- auth state
- last message time
- last pong time
- reconnect count
- last close code/reason
- last error

### 5. Event timeline

A realtime terminal-style feed showing:

- timestamp
- severity
- source
- event type
- message

It should support filtering by category such as `critical`, `jobs`, `browser`, `feeds`, `actions`.

This is the authoritative operational story.

### 6. Manual controls

Even without remote takeover, the user should have safe operational controls:

- pause worker
- resume worker
- restart browser
- restart feed monitors
- acknowledge alert
- force screenshot refresh
- optionally open latest detected job in the worker again

These controls should be grouped clearly and safety-sensitive operations should be visually separated.

## Suggested navigation

Use a watcher operations area with subviews such as:

- `Overview`
- `Timeline`
- `Browser`
- `Jobs`
- `Settings`

The `Overview` page should be sufficient for routine operation. The others provide detail and investigation tooling.

## Notification Model

The first recovery version should deliver alerts and updates to connected clients through the existing per-user realtime websocket path.

Notification types should include:

- informative worker lifecycle updates
- high-value job detected
- job opening started/succeeded/failed
- acceptance started/succeeded/failed
- critical blocked states

Critical alerts must be visually and semantically distinct from normal activity. A captcha or suspicious-login event should not look like an ordinary feed log line.

## Persistence and Storage

For the single-server recovery release, store:

- worker runtime state in the application database
- event timeline in the application database or an event-oriented table store
- lightweight fanout through Redis pub/sub
- browser profiles on disk under a dedicated managed storage path
- screenshots on disk with references in the database

Directory boundaries should make later split-out possible, but the first release should not depend on distributed storage.

## Operational Boundaries

To keep the architecture clean and maintainable:

- Go owns all product logic and state
- the browser is an external supervised process, not a second application boundary
- browser interactions flow through explicit Go interfaces
- the frontend never infers system truth from local heuristics when the backend already knows the answer

This creates a clean break from the Python app rather than reproducing its blurred ownership model.

## Recovery Phases

## Phase 1: Stop the bleeding

Deliver:

- long-lived per-user browser worker
- profile import seeding
- RSS monitor
- Gengo websocket monitor
- runtime snapshots and event timeline
- browser screenshot panel
- job page opening flow
- critical hard-stop alerts
- watcher operations dashboard

This phase restores reliable detection, visibility, and action readiness.

## Phase 2: Safe acceptance

Deliver:

- serialized action coordinator
- accept click flow on the known page layout
- result classification
- richer artifacts around action attempts
- tighter dashboard status around action outcomes

This phase restores the revenue-critical automated acceptance path.

## Phase 3: Hardening and onboarding

Deliver:

- first-run server-hosted login flow for future users
- stronger worker restart and persistence tooling
- richer notification delivery
- multi-user operational improvements
- optional later extensions such as remote takeover or alternative browser backends

## Recommended Build Order

The correct build order is:

1. observability and worker state model
2. browser worker lifecycle and profile import
3. feed monitors and unified event emission
4. browser page open flow
5. dashboard integration
6. safe accept flow

This sequence ensures the team can see what the system is doing before trusting it with money-critical actions.

## Acceptance Criteria for Recovery

The recovery design should be considered successful when all of the following are true:

- the system can maintain a long-lived per-user server-owned browser worker
- the system can show, in the UI, whether RSS, realtime websocket, browser, and actions are healthy independently
- a matching job causes a visible, traceable browser open flow in the worker
- critical safety events immediately block action execution and raise a prominent alert
- the operator can determine from the dashboard exactly what is happening now and why
- the system can progress toward worker-side acceptance without requiring a return to the Python application architecture
