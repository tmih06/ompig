"""host-wiring-run-mode.py is the Python twin of host-wiring-run-mode.mjs: it records the ctx.mode its session_start handler reads."""
import json
import os

import ompig_sdk


def new_extension() -> ompig_sdk.Extension:
    ext = ompig_sdk.Extension("host-wiring-run-mode")

    def on_start(ctx, data):
        # newline="" keeps the "\n" record separator on Windows, where text mode writes "\r\n".
        with open(os.environ["WIRING_REPORT"], "a", newline="") as report:
            report.write(json.dumps({"event": "mode", "value": ctx.mode}, separators=(",", ":")) + "\n")

    ext.on_event("session_start", on_start)
    return ext
