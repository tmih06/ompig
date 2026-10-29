"""Pi core/extensions/types.ts:1315-1338: each tool_result guard is ``e.toolName === "<tool>"``."""

from __future__ import annotations

import ompig_sdk

GUARDS = {
    "bash": ompig_sdk.is_bash_tool_result,
    "powershell": ompig_sdk.is_powershell_tool_result,
    "read": ompig_sdk.is_read_tool_result,
    "edit": ompig_sdk.is_edit_tool_result,
    "write": ompig_sdk.is_write_tool_result,
    "grep": ompig_sdk.is_grep_tool_result,
    "find": ompig_sdk.is_find_tool_result,
    "ls": ompig_sdk.is_ls_tool_result,
}


def test_each_guard_is_true_for_its_own_tool_only() -> None:
    for tool in GUARDS:
        event = {"type": "tool_result", "toolName": tool}
        for other, guard in GUARDS.items():
            assert guard(event) is (other == tool), (other, tool)


def test_guards_reject_custom_tools_and_payloads_without_a_string_tool_name() -> None:
    for other, guard in GUARDS.items():
        for payload in ({"toolName": "my-tool"}, {"type": "tool_result"}, {"toolName": 7}, None):
            assert guard(payload) is False, (other, payload)
