# PRD

## Original problem statement
I want to finish this application and prepare it for launch

## User choices
- Make the current app fully working end-to-end, then improve UI/UX, then fix bugs/stability, then prepare dashboard/admin flows if they exist
- Launch should support both public pages and user accounts
- Add payments
- Keep structure ready because final content will be provided later
- Priority: a bug-free and reliable app

## Architecture decisions
- Kept the existing Next.js frontend and Go backend rather than rewriting the product
- Added a Python FastAPI bridge at `/app/backend/server.py` so the custom Go backend can run inside this environment and proxy through the platform backend service
- Installed PostgreSQL, Redis, and Go 1.25.4 in the container, then configured the bridge to start and proxy the Go API
- Added Stripe billing through the backend bridge using the environment Stripe test key and a `payment_transactions` PostgreSQL table
- Switched the frontend API client to same-origin by default and exposed billing/status flows on `/api/v1/billing/*`
- Routed realtime WebSocket traffic through `/api/ws` so preview traffic reaches the backend correctly

## What has been implemented
- Frontend startup fixed: dependencies installed with modern Yarn, production build passing, app served successfully in preview
- Backend startup fixed: Go service now boots behind the Python bridge with PostgreSQL + Redis available
- Public launch pages polished: upgraded landing page, added pricing page, improved navigation, and linked billing paths
- Reliable auth flow: email/password registration and login working from the live preview, dashboard redirect verified
- Dashboard navigation improved with Dashboard / Translations / Billing / Settings links
- Settings page extended with a billing section and pricing access
- Stripe checkout session creation implemented with server-side fixed plan pricing and status polling endpoint
- Billing status endpoint fixed to return structured JSON, including stored transaction fallback and serialized timestamps
- Health endpoint exposed at `/api/health`
- Realtime websocket URL fixed for preview by routing through `/api/ws`; dashboard shows connected status in preview

## Prioritized backlog
### P0
- Replace the deprecated `middleware.ts` convention with the newer Next.js proxy convention when file deletion/rename is convenient
- Add a true account/subscription sync step after successful Stripe payment (for example setting a subscription tier on the user model)

### P1
- Re-enable social login and magic link only after provider/email credentials are configured for launch
- Add richer billing management (current plan summary, past transactions, billing portal)
- Add a public success/cancel confirmation state on pricing after completing Stripe checkout

### P2
- Add admin-facing management UI if launch scope expands to operations/admin workflows
- Improve dashboard realtime event richness beyond connection state
- Add content polish once final copy/images are supplied

## Next tasks
- Connect successful Stripe payments to a user subscription state in the product data model
- Add a billing history/plan management area in settings
- Bring back OAuth and magic link behind real provider/email configuration
