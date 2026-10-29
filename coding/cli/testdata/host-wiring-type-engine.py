"""host-wiring-type-engine.py computes a schema with a member of Type that the host's
type engine runs (Pick), while its factory runs, and reports the keys it kept.

The Python SDK sends the call to the host (the type.engine request), so the
extension loads only when the ompig binary that hosts it registered the engine."""
import json
import os

import ompig_sdk
from ompig_sdk import Type


def _log(record):
    # newline="" keeps the "\n" record separator on Windows, where text mode writes "\r\n".
    with open(os.environ["WIRING_REPORT"], "a", newline="") as report:
        report.write(json.dumps(record, separators=(",", ":")) + "\n")


def new_extension() -> ompig_sdk.Extension:
    ext = ompig_sdk.Extension("host-wiring-type-engine")
    picked = Type.pick(Type.object({"id": Type.string(), "name": Type.string(), "n": Type.integer()}), ["id", "n"])
    _log({"event": "type_engine", "keys": sorted(picked["properties"])})
    return ext
