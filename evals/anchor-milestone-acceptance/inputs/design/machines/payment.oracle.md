# payment machine -- transition oracle (GENERATED, do not hand-edit)

| test id | stable id | from | event | guard | to | actions |
|---|---|---|---|---|---|---|
| T-PAY-01 | PAY-3f9c21 | draft | authorize | funds_available | authorized | reserve_funds |
| T-PAY-04 | PAY-1a77de | authorized | capture | within_window | captured | post_ledger_entry |
| T-PAY-07 | PAY-b70e44 | captured | settle | batch_open | settled | emit_settlement |
