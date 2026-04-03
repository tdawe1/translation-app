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
- Kept the backend as a single Go Fiber service instead of introducing a second Python server layer
- Added Stripe billing directly to the Go backend using the environment Stripe key and a `payment_transactions` PostgreSQL table
- Switched the frontend API client to same-origin by default and exposed billing/status flows on `/api/v1/billing/*`
- Added Go-native aliases for `/api/health` and `/api/ws` so preview and same-origin frontend traffic keep working without a proxy bridge

## What has been implemented
- Frontend startup fixed: dependencies installed with modern Yarn, production build passing, app served successfully in preview
- Backend startup fixed: Go service now owns the required HTTP, websocket, and billing routes directly
- Public launch pages polished: upgraded landing page, added pricing page, improved navigation, and linked billing paths
- Reliable auth flow: email/password registration and login working from the live preview, dashboard redirect verified
- Dashboard navigation improved with Dashboard / Translations / Billing / Settings links
- Settings page extended with a billing section and pricing access
- Stripe checkout session creation implemented in Go with server-side fixed plan pricing and status polling endpoint
- Billing status endpoint fixed to return structured JSON, including stored transaction fallback and serialized timestamps
- Health endpoint exposed at `/api/health`
- Realtime websocket URL fixed for preview by serving `/api/ws` from the Go backend; dashboard shows connected status in preview

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
