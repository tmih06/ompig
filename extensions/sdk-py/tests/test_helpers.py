"""Pi's truncation helpers and host configuration exports (truncate.ts, config.ts).

The cross-SDK conformance rows run every case against Pi's own code in the Node runtime; these tests keep the documented behavior local. The file mutation queue's
SDK side is in test_file_mutation_queue.py: the queue is the host's.
"""
import json
import os

import pytest

import ompig_sdk


def test_format_size_rounds_half_up_like_to_fixed():
    cases = {0: "0B", 1023: "1023B", 1024: "1.0KB", 1280: "1.3KB", 1536: "1.5KB", 1048575: "1024.0KB", 1048576: "1.0MB", 5505024: "5.3MB"}
    for size, want in cases.items():
        assert ompig_sdk.format_size(size) == want


def test_truncate_tail_keeps_the_end_of_an_oversized_last_line():
    result = ompig_sdk.truncate_tail("日本語のテキスト", max_bytes=7)
    assert (result.content, result.last_line_partial, result.truncated_by, result.output_bytes) == ("スト", True, "bytes", 6)


def test_truncate_head_never_returns_a_partial_line():
    result = ompig_sdk.truncate_head("no newline at all", max_bytes=4)
    assert (result.content, result.first_line_exceeds_limit, result.truncated_by) == ("", True, "bytes")
    result = ompig_sdk.truncate_head("a\nb\nc\n", max_lines=2)
    assert (result.content, result.truncated_by, result.total_lines, result.output_lines) == ("a\nb", "lines", 3, 2)


def test_truncate_limits_of_zero_are_used_as_given_and_none_takes_the_default():
    assert ompig_sdk.truncate_head("a\nb", max_lines=0).content == ""
    untouched = ompig_sdk.truncate_head("a\nb")
    assert (untouched.truncated, untouched.max_lines, untouched.max_bytes) == (False, ompig_sdk.DEFAULT_MAX_LINES, ompig_sdk.DEFAULT_MAX_BYTES)


def test_truncate_line_counts_utf16_units():
    assert ompig_sdk.truncate_line("😀😀😀😀", 6) == ("😀😀😀... [truncated]", True)
    assert ompig_sdk.truncate_line("😀😀😀", 6) == ("😀😀😀", False)
    assert ompig_sdk.truncate_line("x" * 501)[1] is True and ompig_sdk.truncate_line("x" * 500)[1] is False


def _with_host_info(value, check):
    saved = os.environ.get("OMPIG_EXT_HOST_INFO")
    try:
        if value is None:
            os.environ.pop("OMPIG_EXT_HOST_INFO", None)
        else:
            os.environ["OMPIG_EXT_HOST_INFO"] = value
        check()
    finally:
        if saved is None:
            os.environ.pop("OMPIG_EXT_HOST_INFO", None)
        else:
            os.environ["OMPIG_EXT_HOST_INFO"] = saved


def test_host_info_reads_the_hosts_environment_entry():
    def check():
        assert ompig_sdk.get_agent_dir() == "/home/u/.ompig/agent"
        assert ompig_sdk.get_config_dir_name() == ".ompig"
        assert ompig_sdk.get_version() == "1.1.0"

    _with_host_info(json.dumps({"agentDir": "/home/u/.ompig/agent", "configDirName": ".ompig", "version": "1.1.0"}), check)


@pytest.mark.parametrize("value", [None, "", "{", "[]", json.dumps({"configDirName": ".ompig"})])
def test_host_info_is_unavailable_without_a_host_that_provides_it(value):
    def check():
        with pytest.raises(ompig_sdk.HostInfoUnavailable):
            ompig_sdk.get_agent_dir()

    _with_host_info(value, check)
