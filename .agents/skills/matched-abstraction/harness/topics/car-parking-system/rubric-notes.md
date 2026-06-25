# Topic Notes: Car Parking System

Use these notes to judge topic fit. They are not a golden answer.

Strong plans usually cover:

- Facility, level, zone, spot, gate, kiosk, camera, permit, reservation, payment, and parking session concepts.
- Entry flow, exit flow, reservation arrival, permit validation, lost ticket or unreadable plate flow, payment failure, and manual attendant override.
- Occupancy reconciliation when sensors, cameras, gates, and attendants disagree.
- Pricing and rule configuration as owned domain behavior, not hardcoded task detail.
- External boundaries for payment provider, camera or LPR service, gate controller, kiosk, notification service, and reporting exports.
- Data integrity concerns such as idempotent hardware events, duplicate plates, stale occupancy, and audit trails.
- Operational views for attendants, facility managers, and operations staff.
- Verification strategy covering state transitions, event ordering, pricing examples, hardware simulation, payment failure, and reconciliation.
- A fit artifact set such as a domain concept map, hardware boundary diagram, entry/exit sequence flows, session or occupancy state model, pricing rule examples, reconciliation matrix, and verification matrix. Strong outputs explain which artifacts belong now, which should be deferred, and what later work must match.

Common gaps:

- Treating parking as only a CRUD app for spots.
- Omitting offline or degraded hardware behavior.
- Skipping auditability for manual overrides and payment/session changes.
- Listing every possible diagram without explaining distinct derive/match value or depth.
- Producing implementation tasks before proving the Plan boundary.
