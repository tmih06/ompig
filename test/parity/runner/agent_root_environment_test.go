//go:build parity

// SPDX-FileCopyrightText: Copyright Hewlett Packard Enterprise Development LP
// SPDX-License-Identifier: MIT

package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

var agentRootEnvironmentKeys = []string{"OMPIG_HOME", "OMPIG_CODING_AGENT_DIR", "OMPIG_CODING_AGENT_SESSION_DIR", "PI_CODING_AGENT_DIR"}

func TestParityEnvironmentChild(t *testing.T) {
	if os.Getenv("OMPIG_TEST_ENV_HELPER") != "1" {
		return
	}
	values := map[string]string{}
	for _, key := range agentRootEnvironmentKeys {
		values[key] = os.Getenv(key)
	}
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(data))
	os.Exit(0)
}

func TestHermeticCommandsDoNotInheritWorkerAgentRoots(t *testing.T) {
	for _, key := range agentRootEnvironmentKeys {
		t.Setenv(key, "ambient-agent-path")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	output, exit, _, err := runCmd(t.Context(), executable, []string{"-test.run=^TestParityEnvironmentChild$"}, []string{"OMPIG_TEST_ENV_HELPER=1", "OMPIG_HOME=fixture-home"}, 5*time.Second, t.TempDir())
	if err != nil || exit != 0 {
		t.Fatalf("exit=%d err=%v output=%s", exit, err, output)
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(output), &values); err != nil {
		t.Fatal(err)
	}
	for _, key := range agentRootEnvironmentKeys {
		want := ""
		if key == "OMPIG_HOME" {
			want = "fixture-home"
		}
		if values[key] != want {
			t.Fatalf("%s=%q, want %q", key, values[key], want)
		}
	}
	output, exit, _, err = runCmd(t.Context(), executable, []string{"-test.run=^TestParityEnvironmentChild$"}, []string{"OMPIG_TEST_ENV_HELPER=1", "OMPIG_CODING_AGENT_DIR=explicit-fixture-agent"}, 5*time.Second, t.TempDir())
	if err != nil || exit != 0 {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
	if err := json.Unmarshal([]byte(output), &values); err != nil {
		t.Fatal(err)
	}
	if values["OMPIG_CODING_AGENT_DIR"] != "explicit-fixture-agent" {
		t.Fatal(values)
	}
}

func TestTmuxPrefixClearsWorkerAgentRoots(t *testing.T) {
	prefix := tmuxEnvPrefix("")
	for _, key := range agentRootEnvironmentKeys {
		if !strings.Contains(strings.Split(prefix, ";")[0], key) {
			t.Errorf("tmux prefix retains %s: %s", key, prefix)
		}
	}
}
