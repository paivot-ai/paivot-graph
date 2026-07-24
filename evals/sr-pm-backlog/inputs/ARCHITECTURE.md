# ARCHITECTURE.md: CurbSide Parking Reservations

## Stack

- Frontend: React SPA (Vite), talks to the API over JSON.
- API: Node.js with Express.
- Database: PostgreSQL 15. Single instance for MVP.
- Auth: email/password, bcrypt cost 12, JWT bearer tokens with 30-minute
  expiry. Header: `Authorization: Bearer <token>`.
- Local dev and pilot deployment via a single docker-compose file.

## Data model (PostgreSQL)

- `users` (id UUID PK, email VARCHAR UNIQUE, password_hash VARCHAR,
  role VARCHAR CHECK IN ('driver','operator'), created_at TIMESTAMPTZ)
- `garages` (id UUID PK, operator_id UUID FK -> users.id, name VARCHAR,
  zone VARCHAR, created_at TIMESTAMPTZ)
- `spots` (id UUID PK, garage_id UUID FK -> garages.id, label VARCHAR,
  hourly_rate_cents INTEGER, is_active BOOLEAN DEFAULT true)
- `reservations` (id UUID PK, spot_id UUID FK -> spots.id,
  driver_id UUID FK -> users.id, starts_at TIMESTAMPTZ, ends_at TIMESTAMPTZ,
  status VARCHAR CHECK IN ('active','cancelled','completed'),
  created_at TIMESTAMPTZ)

## API endpoints

- `POST /api/auth/register` and `POST /api/auth/login`
- `GET /api/garages?zone=` (public search)
- `GET /api/garages/:id/spots?from=&to=` (availability for a window)
- `POST /api/reservations` body `{spotId, startsAt, endsAt}` returns 201
- `DELETE /api/reservations/:id` cancels (sets status to `cancelled`),
  returns 204; 403 when the reservation belongs to another driver
- Operator (JWT with role operator): `POST /api/garages`,
  `POST /api/garages/:id/spots`, `PATCH /api/spots/:id`

## Invariants

- No double-booking: an insert into `reservations` that overlaps an existing
  `active` reservation for the same `spot_id` is rejected with 409. Enforced
  in the database (exclusion constraint or serialized transaction), not only
  in application code.
- Passwords are never stored or logged in plaintext.

## Configuration

Environment variables: `DATABASE_URL`, `JWT_SECRET`, `PORT`.

## Testing strategy

- Unit tests for validation and service logic (mocks acceptable here only).
- Integration tests run against a real PostgreSQL instance (docker-compose);
  no mocks in integration tests.
- Coverage target: minimum 80% combined (unit plus integration).
