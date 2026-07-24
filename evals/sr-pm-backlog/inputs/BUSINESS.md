# BUSINESS.md: CurbSide Parking Reservations

## Product summary

CurbSide is a web application that lets drivers reserve a guaranteed parking
spot in small private garages before they arrive, and lets garage operators
list their spots and see utilization. MVP targets a single city.

## Business goals

1. Drivers can find and reserve a spot for a time window in under two minutes.
2. Garage operators can list a garage and its spots without staff assistance.
3. Reach 500 completed reservations per month within 6 months of launch.

## Personas

- **Driver**: commutes into the city 2-4 days a week, wants certainty that a
  spot exists before leaving home.
- **Operator**: owns or manages 1-3 small garages (10-60 spots each), wants
  incremental revenue from idle capacity with minimal admin work.

## Scope (MVP)

In scope:
- Driver account registration and login (email/password)
- Search garages by zone, view available spots and hourly price
- Reserve a spot for a time window; cancel a reservation
- Operator: create a garage, add spots, deactivate a spot
- "My Reservations" list for drivers

Out of scope for MVP:
- Payments and billing (reservations are free during pilot)
- Mobile native apps (responsive web only)
- Dynamic pricing, waitlists, recurring reservations

## Constraints

- Small team; ship the MVP with a simple, boring, well-tested stack.
- Web only, responsive; must work on mobile browsers.
- No special regulatory regime, but account data must be handled with
  standard privacy hygiene (hashed passwords, no plaintext credentials).

## Success metrics

- Reservation conversion: 40% of searches that show availability end in a
  reservation.
- Double-booking incidents: zero. A spot must never be reserved twice for
  overlapping time windows.
- Cancellation flow completion: under 30 seconds from "My Reservations".
