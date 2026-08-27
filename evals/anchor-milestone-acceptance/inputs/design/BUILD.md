# BUILD.md -- Ledger service

## 9. Build plan

**M1 - Payment slice.** Payment lifecycle end to end, authorization enforced.
DoD: the payment transition rows `PAY-3f9c21` and `T-PAY-04` are green by stable id,
the authorization decision row `AUTHZ-a0788c` is covered by the policy conformance
suite, integration tests run against a real ledger with no mocks below the boundary,
and G4-import is clean.

**M2 - Settlement slice.** Settlement lifecycle end to end.
DoD: the settlement transition rows `PAY-b70e44` are green by stable id, and the
reconciliation job has its failure-catalog signals wired.
Status: open
