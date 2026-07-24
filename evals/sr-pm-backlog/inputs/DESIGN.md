# DESIGN.md: CurbSide Parking Reservations

## User journeys

### Driver: first reservation
1. Register with email and password; land on the Search page.
2. Pick a zone and a time window (start, end); see garages with availability.
3. Open a garage; see its available spots with the hourly price.
4. Reserve a spot; see a confirmation screen with garage name, spot label,
   and time window; the reservation appears in "My Reservations".

### Driver: cancellation
1. Open "My Reservations"; each active reservation shows a Cancel button.
2. Confirm cancellation in a dialog; the row updates to Cancelled in place.

### Operator: listing a garage
1. Register as an operator; land on the Operator Dashboard.
2. Create a garage (name, zone); add spots (label, hourly price).
3. Deactivate a spot; it disappears from driver search immediately.

## Screens

- **Search**: zone selector, time-window picker, result list of garages with
  count of available spots. Empty state: "No availability in this zone for
  that window."
- **Garage detail**: spot list (label, hourly price), Reserve button per spot.
- **Confirmation**: summary plus a link to My Reservations.
- **My Reservations**: active reservations first, then past/cancelled.
- **Operator Dashboard**: garages owned, spot management per garage.

## Interaction and error design

- Validation errors display inline below the offending field, in red text.
- A reservation attempt on a spot taken in the meantime shows: "That spot was
  just reserved. Pick another." and refreshes availability.
- Cancelling is always confirmed by a dialog; no silent destructive actions.
- All times are shown in the garage's local timezone with an explicit label.

## Usability requirements

- WCAG 2.1 AA: keyboard navigable, labeled inputs, sufficient contrast.
- Search-to-confirmation in at most 4 screens (per business goal 1).
- Responsive layout; primary flows usable on a 375px-wide viewport.
