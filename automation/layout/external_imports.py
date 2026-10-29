#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
"""Build the real embedding example and four library imports outside Ompig's module tree."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def check(root):
    root = root.resolve()
    caches = json.loads(subprocess.check_output(["go", "env", "-json", "GOPATH", "GOMODCACHE", "GOCACHE"]))
    with tempfile.TemporaryDirectory(prefix="layout-import-") as directory:
        consumer = Path(directory)
        (consumer / "go.mod").write_text(
            "module example.org/layout-consumer\n\ngo 1.26.0\n"
            "require github.com/tmih06/ompig v0.0.0\n"
            f"replace github.com/tmih06/ompig => {json.dumps(str(root))}\n"
            f"replace github.com/tmih06/ompig/extensions/sdk => {json.dumps(str(root / 'extensions/sdk'))}\n"
        )
        shutil.copyfile(root / "examples/sdk/main.go", consumer / "main.go")
        (consumer / "imports.go").write_text('''package main
import (
    "github.com/tmih06/ompig/agent"
    "github.com/tmih06/ompig/ai"
    "github.com/tmih06/ompig/coding"
    "github.com/tmih06/ompig/coding/extension"
    "github.com/tmih06/ompig/coding/extension/host/inproc"
    "github.com/tmih06/ompig/coding/rpcclient"
    "github.com/tmih06/ompig/tui"
    "github.com/tmih06/ompig/tui/widthx"
)
var (
    _ agent.AgentTool
    _ ai.Model
    _ = coding.SessionOptions{Runner: (*inproc.Runner)(nil)}
    _ extension.Extension
    _ = rpcclient.NewRpcClient
    _ = tui.NewText
    _ = widthx.VisibleWidth
)
''')
        env = os.environ | caches | {"GOWORK": "off", "GOPROXY": "off", "HOME": str(consumer / "home"),
                            "OMPIG_CODING_AGENT_DIR": str(consumer / "ompig"),
                            "PI_CODING_AGENT_DIR": str(consumer / "pi")}
        for name in ("home", "ompig", "pi"):
            (consumer / name).mkdir()
        subprocess.run(["go", "build", "-mod=mod", "."], cwd=consumer, env=env, check=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("root", type=Path)
    check(parser.parse_args().root)
