# SPDX-License-Identifier: MIT
"""Harbor agents for the Ompig benchmarks. Needs Harbor 0.24.0 importable (pigbench.harbor.HARBOR_VERSION and HARBOR_WHEEL_SHA256); see ../README.md.

OmpigAgent subclasses Harbor's `Pi` adapter (harbor/agents/installed/pi.py in the Harbor wheel). Harbor's adapter runs
`pi --print --mode json --session-dir <logs>/pi/sessions --provider P --model M <instruction>` and converts the session JSONL
the harness wrote into a trajectory and token accounting (`usage.input/output/cacheRead/cacheWrite/reasoning/totalTokens/cost`).
For a custom endpoint (the metering gateway) it writes models.json into the directory named by PI_CODING_AGENT_DIR. Ompig reads its
own OMPIG_CODING_AGENT_DIR instead (Ompig divergence D2) unless OMPIG_USE_PI_DIRS=1 selects Pi's directories, so OmpigAgent sets
OMPIG_USE_PI_DIRS=1 on every command; without it Ompig fails with `Unknown provider "harbor-endpoint"`. Otherwise Ompig accepts the same
flags and writes the same session and usage fields (checked by `pigbench harbor-compat`), so OmpigAgent puts a pinned Ompig binary on
PATH under the name `pi` and the inherited command line and converter run unchanged. Nothing is patched in Harbor.

CopilotByokAgent subclasses Harbor's `CopilotCli`. Harbor's adapter forwards only a GitHub token to Copilot CLI
(copilot_cli.py, `run`), so bring-your-own-key mode has no passthrough. CopilotByokAgent adds the BYOK variables
(COPILOT_PROVIDER_BASE_URL, COPILOT_PROVIDER_TYPE, COPILOT_PROVIDER_API_KEY, COPILOT_MODEL, COPILOT_PROVIDER_MAX_PROMPT_TOKENS,
COPILOT_PROVIDER_MAX_OUTPUT_TOKENS, COPILOT_OFFLINE) from the host environment to every command the adapter runs. The key is never
written to disk or to a config file.
"""
import os
import shlex
from typing import override

from harbor.agents.installed.copilot_cli import CopilotCli
from harbor.agents.installed.pi import Pi
from harbor.environments.base import BaseEnvironment

BYOK_VARIABLES = ("COPILOT_PROVIDER_BASE_URL", "COPILOT_PROVIDER_TYPE", "COPILOT_PROVIDER_API_KEY", "COPILOT_MODEL", "COPILOT_PROVIDER_MAX_PROMPT_TOKENS",
                  "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS", "COPILOT_OFFLINE")


# Ompig divergence D2: Ompig reads OMPIG_CODING_AGENT_DIR unless this selects Pi's PI_CODING_AGENT_DIR, which Harbor's Pi adapter sets.
OMPIG_ENV = {"OMPIG_USE_PI_DIRS": "1"}


def byok_env(environ=None):
    """The BYOK variables set on the host, as a dict. Missing variables are left out."""
    environ = os.environ if environ is None else environ
    return {k: environ[k] for k in BYOK_VARIABLES if environ.get(k)}


class OmpigAgent(Pi):
    """Ompig run through Harbor's Pi adapter. Select it with `--agent-import-path pigharbor.agents:OmpigAgent`.

    agent kwargs: `ompig_url` and `ompig_sha256` (a pinned Ompig release archive, checked in the container before it is unpacked) or `ompig_binary`
    (a host path to a static Linux build that is uploaded). Exactly one source is required.
    """

    def __init__(self, *args, ompig_url: str | None = None, ompig_sha256: str | None = None, ompig_binary: str | None = None, **kwargs):
        if bool(ompig_url) == bool(ompig_binary):
            raise ValueError("OmpigAgent needs exactly one of ompig_url (with ompig_sha256) or ompig_binary")
        if ompig_url and not ompig_sha256:
            raise ValueError("ompig_url needs ompig_sha256: a Ompig archive is never installed unchecked")
        super().__init__(*args, **kwargs)
        self._ompig_url, self._ompig_sha256, self._ompig_binary = ompig_url, ompig_sha256, ompig_binary

    @staticmethod
    @override
    def name() -> str:
        return "ompig"

    @override
    def get_version_command(self) -> str | None:
        return "ompig --version"

    @override
    async def install(self, environment: BaseEnvironment) -> None:
        await self.ensure_system_dependencies(environment, ("curl", "ca_certificates"))
        if self._ompig_binary:
            await environment.upload_file(source_path=self._ompig_binary, target_path="/tmp/ompig")
            fetch = "install -m 0755 /tmp/ompig /usr/local/bin/ompig"
        else:
            fetch = (f"curl -fsSL -o /tmp/ompig.tar.gz {shlex.quote(self._ompig_url)} && echo {shlex.quote(self._ompig_sha256 + '  /tmp/ompig.tar.gz')} | sha256sum -c - "
                     "&& rm -rf /tmp/ompig-release && mkdir -p /tmp/ompig-release && tar -xzf /tmp/ompig.tar.gz -C /tmp/ompig-release "
                     "&& install -m 0755 \"$(find /tmp/ompig-release -type f -name ompig | head -1)\" /usr/local/bin/ompig")
        await self.exec_as_root(environment, command=f"set -euo pipefail; {fetch} && ln -sf /usr/local/bin/ompig /usr/local/bin/pi && ompig --version")

    @override
    async def exec_as_agent(self, environment, command, env=None, cwd=None, timeout_sec=None):
        return await super().exec_as_agent(environment, command=command, env={**OMPIG_ENV, **(env or {})}, cwd=cwd, timeout_sec=timeout_sec)


class CopilotByokAgent(CopilotCli):
    """Copilot CLI in bring-your-own-key mode. Select it with `--agent-import-path pigharbor.agents:CopilotByokAgent`."""

    @staticmethod
    @override
    def name() -> str:
        return "copilot-byok"

    @override
    async def exec_as_agent(self, environment, command, env=None, cwd=None, timeout_sec=None):
        return await super().exec_as_agent(environment, command=command, env={**byok_env(), **(env or {})}, cwd=cwd, timeout_sec=timeout_sec)
