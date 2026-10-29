package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	ompiglet "github.com/tmih06/ompig/coding/ompiglet"
)

// TestPigletStripRemovesPigletCommand runs `ompig --mode rpc --ompiglet` with a Ompiglet that keeps /ompiglet and one that strips
// it (strip.commands: /ompiglet, D92). The keeping Ompiglet lists /ompiglet in RPC get_commands and the extension API's
// getCommands, and a typed /ompiglet runs the command. The stripping Ompiglet lists it nowhere, and a typed /ompiglet goes to
// the model as the user's text, as an unknown slash command does in Pi (Pi has no /ompiglet).
func TestPigletStripRemovesPigletCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("runs ompig --mode rpc")
	}
	for _, tc := range []struct {
		name, yaml string
		stripped   bool
	}{
		{name: "keep", yaml: "name: lean\n"},
		{name: "strip", yaml: "name: lean\nstrip:\n  commands: [/ompiglet]\n", stripped: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			home, cwd := t.TempDir(), t.TempDir()
			path := filepath.Join(t.TempDir(), "lean.yaml")
			probe := filepath.Join(cwd, "probe.mjs")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(probe, []byte(stripLeakProbe), 0o600); err != nil {
				t.Fatal(err)
			}
			env := []string{"HOME=" + home, "OMPIG_HOME=" + home, "OMPIG_CODING_AGENT_DIR=" + filepath.Join(home, "agent"), "OMPIG_TEST_FAUX=1", "OMPIG_OFFLINE=1", "NO_COLOR=1"}
			p := startRPCProcessAt(t, cwd, env, "--no-session", "--no-context-files", "--no-extensions", "--provider", "test-faux", "--model", "faux-1", "-e", probe, "--ompiglet", path)

			p.send(`{"id":"commands","type":"get_commands"}`)
			var rpcCommands []string
			p.await("get_commands", func(record rpcRecord) bool {
				if record["type"] != "response" || record["id"] != "commands" {
					return false
				}
				data, _ := record["data"].(map[string]any)
				commands, _ := data["commands"].([]any)
				for _, value := range commands {
					command, _ := value.(map[string]any)
					name, _ := command["name"].(string)
					rpcCommands = append(rpcCommands, name)
				}
				return true
			})
			if got := slices.Contains(rpcCommands, ompiglet.InspectCommand); got == tc.stripped {
				t.Errorf("RPC get_commands lists /ompiglet = %v under %s: %v", got, tc.name, rpcCommands)
			}

			p.send(`{"id":"dump","type":"prompt","message":"/leak-dump"}`)
			p.await("extension API getCommands", func(record rpcRecord) bool {
				if record["type"] != "extension_ui_request" || record["method"] != "notify" {
					return false
				}
				message, _ := record["message"].(string)
				payload, ok := strings.CutPrefix(message, "LEAKDUMP:")
				if !ok {
					return false
				}
				var dump map[string]string
				if err := json.Unmarshal([]byte(payload), &dump); err != nil {
					t.Fatalf("probe dump %q: %v", payload, err)
				}
				found := false
				for line := range strings.SplitSeq(dump["getCommands"], "\n") {
					if name, _, _ := strings.Cut(line, " "); name == "/"+ompiglet.InspectCommand {
						found = true
					}
				}
				if found == tc.stripped {
					t.Errorf("getCommands lists /ompiglet = %v under %s:\n%s", found, tc.name, dump["getCommands"])
				}
				return true
			})

			// /ompiglet prints "Ompiglet: <name>" and then the release and source lines.
			pigletOutput := func(record rpcRecord) bool {
				message, _ := record["message"].(string)
				return record["type"] == "extension_ui_request" && record["method"] == "notify" && strings.HasPrefix(message, "Ompiglet: lean\n")
			}
			p.send(`{"id":"typed","type":"prompt","message":"/ompiglet"}`)
			if !tc.stripped {
				p.await("/ompiglet output", pigletOutput)
				p.closeAndWait("after /ompiglet")
				return
			}
			p.await("the model turn of typed /ompiglet", func(record rpcRecord) bool {
				if pigletOutput(record) {
					t.Fatal("stripped /ompiglet still ran the command")
				}
				return record["type"] == "agent_settled"
			})
			p.send(`{"id":"history","type":"get_messages"}`)
			p.await("get_messages", func(record rpcRecord) bool {
				if record["type"] != "response" || record["id"] != "history" {
					return false
				}
				encoded, _ := json.Marshal(record["data"])
				type part struct{ Type, Text string }
				var data struct {
					Messages []struct {
						Role    string
						Content json.RawMessage
					}
				}
				if err := json.Unmarshal(encoded, &data); err != nil {
					t.Fatal(err)
				}
				var roles []string
				var user []part
				for _, message := range data.Messages {
					roles = append(roles, message.Role)
					if message.Role == "user" {
						user = nil
						if err := json.Unmarshal(message.Content, &user); err != nil {
							t.Fatalf("user content %s: %v", message.Content, err)
						}
					}
				}
				if want := []part{{Type: "text", Text: "/ompiglet"}}; !slices.Equal(user, want) {
					t.Errorf("user message of typed /ompiglet = %+v, want %+v (roles %v)", user, want, roles)
				}
				if !slices.Contains(roles, "assistant") {
					t.Errorf("typed /ompiglet got no model answer: roles %v", roles)
				}
				return true
			})
			p.closeAndWait("after typed /ompiglet")
		})
	}
}
