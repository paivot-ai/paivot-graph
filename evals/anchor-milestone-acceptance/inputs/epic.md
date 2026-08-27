# LEDG-e1 (milestone epic, label: milestone, accepted slices: LEDG-e2, LEDG-e3)

BUILD PLAN MILESTONE: M1

DoD: the payment transition rows `PAY-3f9c21` and `T-PAY-04` are green by stable id,
the authorization decision row `AUTHZ-a0788c` is covered by the policy conformance
suite, integration tests run against a real ledger with no mocks below the boundary,
and G4-import is clean.

Slice epics (both closed and accepted through their own completion gates):

- LEDG-e2 Authorize + reserve funds (walking skeleton, 6 stories)
- LEDG-e3 Capture + ledger posting (5 stories)
