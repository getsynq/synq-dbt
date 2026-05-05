#!/usr/bin/env python3
"""
Probe that simulates the dbt-snowflake signal-handling contract.

The wrapper (synq-dbt) is supposed to deliver SIGINT to its child on
cancellation so a long-running query can be cancelled cleanly. This script
registers a SIGINT/SIGTERM handler that flips an in-memory cursor's state
to "cancelled" before exiting — exactly what
snowflake.connector.cursor.cancel() would do over the wire when dbt-core
runs its KeyboardInterrupt handler.

We deliberately avoid importing snowflake-connector here so the probe runs
on any plain Linux/macOS machine without network access. End-to-end
validation against a real Snowflake account belongs in a build-tagged
integration test that we can run out-of-band.

Output protocol (single line each, all lines flushed immediately):
  STARTED                  -- handler registered, "query" issued
  SIGNAL_RECEIVED=<num>    -- handler fired (proves wrapper sent a signal,
                              not SIGKILL)
  FINAL_STATE=<state>      -- cursor.cancel() ran; "cancelled" is good
  COMPLETED state=<state>  -- ONLY printed if the query ran to completion
                              despite cancellation -- this is the bug
"""
import signal
import sys
import time


class FakeCursor:
    def __init__(self):
        self.state = "idle"

    def execute_long(self):
        self.state = "running"
        # Simulates a long Snowflake query, e.g. CALL SYSTEM$WAIT(60, 'SECONDS').
        time.sleep(60)
        self.state = "completed"

    def cancel(self):
        # Real cursor.cancel() sends "ABORT QUERY <id>" to Snowflake.
        self.state = "cancelled"


cur = FakeCursor()


def on_signal(signum, frame):
    print(f"SIGNAL_RECEIVED={signum}", flush=True)
    cur.cancel()
    print(f"FINAL_STATE={cur.state}", flush=True)
    sys.exit(130)  # 128 + SIGINT(2), conventional shell exit for Ctrl-C


signal.signal(signal.SIGINT, on_signal)
signal.signal(signal.SIGTERM, on_signal)

print("STARTED", flush=True)
cur.execute_long()
# We must NOT reach this line on a cancelled run.
print(f"COMPLETED state={cur.state}", flush=True)
