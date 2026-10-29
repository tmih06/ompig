package storage_test

// pi: packages/durable/src/storage/jsonl/storage.ts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tmih06/ompig/durable/storage/jsonl"
	jsonlnode "github.com/tmih06/ompig/durable/storage/jsonl/node"
	"github.com/tmih06/ompig/internal/testenv"
)

func runPiJSONL(t *testing.T, mode string, programs []oracleProgram, dirs []string) [][]oracleResult {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"programs": programs, "dirs": dirs})
	if err != nil {
		t.Fatal(err)
	}
	root := testenv.ModuleRoot(t)
	cmd := exec.CommandContext(t.Context(), "node", "--disable-warning=ExperimentalWarning", "testdata/memory_oracle.mjs", filepath.Join(root, ".upstream", "current", "packages"), mode)
	cmd.Stdin = bytes.NewReader(payload)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Pi oracle (%s): %v\n%s", mode, err, &stderr)
	}
	var results [][]oracleResult
	if err := json.Unmarshal(output, &results); err != nil {
		t.Fatal(err)
	}
	return results
}

func runPigJSONL(t *testing.T, programs []oracleProgram, dirs []string, readsOnly bool) [][]oracleResult {
	t.Helper()
	out := make([][]oracleResult, len(programs))
	for i, program := range programs {
		store, err := jsonlnode.OpenNodeJsonlStorage(context.Background(), dirs[i], jsonl.JsonlStorageOptions{})
		if err != nil {
			name := "Error"
			if named, ok := err.(interface{ Name() string }); ok {
				name = named.Name()
			}
			out[i] = []oracleResult{{Index: -1, Err: &struct {
				Name    string `json:"name"`
				Message string `json:"message"`
			}{name, err.Error()}}}
			continue
		}
		out[i] = runPigProgram(t, program, store, readsOnly)
		if err := store.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// jsonlLines reads a storage directory as {file: [decoded line]}, so files compare by content without depending on member order.
func jsonlLines(t *testing.T, dir string) map[string][]any {
	t.Helper()
	files := map[string][]any{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var lines []any
		for line := range strings.SplitSeq(strings.TrimSuffix(string(raw), "\n"), "\n") {
			var decoded any
			if err := json.Unmarshal([]byte(line), &decoded); err != nil {
				t.Fatalf("%s: %q: %v", entry.Name(), line, err)
			}
			lines = append(lines, decoded)
		}
		files[entry.Name()] = lines
	}
	return files
}

// Pi's JSONL storage and Ompig's run the same seeded programs on their own directories, then each reads its own directory and a copy of the other's: the answers, the files each writer left, the files after each recovery and the state read back must all agree.
func TestJsonlStorageMatchesPiOnSeededPrograms(t *testing.T) {
	const seeded = 150
	programs := make([]oracleProgram, seeded)
	for i := range programs {
		programs[i] = oracleProgramFor(int64(20262020+i), true)
	}
	programs = append(programs, requestKeyPrograms()...)
	programs = append(programs, documentCopyPrograms()...)
	programs = append(programs, danglingParentPrograms()...)
	programs = append(programs, documentBoundaryPrograms()...)
	programs = append(programs, indexedStringPrograms()...)
	base := t.TempDir()
	piDirs, ompigDirs := make([]string, len(programs)), make([]string, len(programs))
	for i := range programs {
		piDirs[i], ompigDirs[i] = filepath.Join(base, fmt.Sprintf("pi-%d", i)), filepath.Join(base, fmt.Sprintf("ompig-%d", i))
	}

	piWrite := runPiJSONL(t, "jsonl-write", programs, piDirs)
	ompigWrite := runPigJSONL(t, programs, ompigDirs, false)
	ops, failures, differing := map[string]int{}, 0, 0
	for i, program := range programs {
		differing += compareOracleResults(t, fmt.Sprintf("write program %d", i), program, ompigWrite[i], piWrite[i], differing, ops, &failures)
	}

	// What each writer left, before any reopen: recovery reclaims superseded sidecar records and would hide a writer that never did.
	for i := range programs {
		differing += compareJSONLDirs(t, "after the writes", i, piDirs[i], ompigDirs[i], differing)
	}

	// Each reader opens its own copy, so no reader's recovery changes the files another reader starts from.
	piOfPig, ompigOfPi := copyJSONLDirs(t, ompigDirs, base, "pi-reads-ompig"), copyJSONLDirs(t, piDirs, base, "ompig-reads-pi")
	piReadsPi := runPiJSONL(t, "jsonl-read", programs, piDirs)
	piReadsPig := runPiJSONL(t, "jsonl-read", programs, piOfPig)
	ompigReadsPi := runPigJSONL(t, programs, ompigOfPi, true)
	ompigReadsPig := runPigJSONL(t, programs, ompigDirs, true)
	for i := range programs {
		if piReadsPi[i][0].Index < 0 {
			t.Errorf("program %d: Pi cannot reopen its own directory: %s", i, piReadsPi[i][0].Err.Message)
		}
	}
	for i, program := range programs {
		for label, got := range map[string][]oracleResult{"Pi reads Ompig's directory": piReadsPig[i], "Ompig reads Pi's directory": ompigReadsPi[i], "Ompig reads its own directory": ompigReadsPig[i]} {
			differing += compareOracleResults(t, fmt.Sprintf("%s, program %d", label, i), program, got, piReadsPi[i], differing, map[string]int{}, new(int))
		}
		// Recovery reclaims the same records: each storage's own directory, and each storage's copy of the other's.
		differing += compareJSONLDirs(t, "after reopening its own directory", i, piDirs[i], ompigDirs[i], differing)
		differing += compareJSONLDirs(t, "after reopening the other's directory", i, ompigOfPi[i], piOfPig[i], differing)
	}
	if differing > 8 {
		t.Errorf("%d answers differ", differing)
	}
	if failures < 20 || ops["commit"] < 300 || ops["scanEntries"] < 50 {
		t.Errorf("probes are one-sided: %v, %d rejections", ops, failures)
	}
}

// compareJSONLDirs compares two storage directories at one stage: every file's decoded records, and main.jsonl byte for byte (compareJSONLBytes).
func compareJSONLDirs(t *testing.T, stage string, program int, piDir, ompigDir string, reported int) int {
	t.Helper()
	differing := compareJSONLBytes(t, piDir, ompigDir, program, reported)
	if piFiles, ompigFiles := jsonlLines(t, piDir), jsonlLines(t, ompigDir); !reflect.DeepEqual(piFiles, ompigFiles) {
		if differing++; reported+differing <= 8 {
			t.Errorf("program %d, %s: persisted records differ\n  Ompig %v\n  Pi  %v", program, stage, ompigFiles, piFiles)
		}
	}
	return differing
}

// copyJSONLDirs copies each storage directory to a fresh one under base.
func copyJSONLDirs(t *testing.T, dirs []string, base, label string) []string {
	t.Helper()
	copies := make([]string, len(dirs))
	for i, dir := range dirs {
		copies[i] = filepath.Join(base, fmt.Sprintf("%s-%d", label, i))
		if err := os.CopyFS(copies[i], os.DirFS(dir)); err != nil {
			t.Fatal(err)
		}
	}
	return copies
}

// compareJSONLBytes compares the persisted lines byte for byte, as Pi's JSON.stringify wrote them, except records whose member order the generator
// does not control (tasks and submissions list their members in Pi's order of construction, which a seeded generator does not reproduce).
func compareJSONLBytes(t *testing.T, piDir, ompigDir string, program, reported int) int {
	t.Helper()
	differing := 0
	entries, err := os.ReadDir(piDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		a, _ := os.ReadFile(filepath.Join(piDir, entry.Name()))
		b, _ := os.ReadFile(filepath.Join(ompigDir, entry.Name()))
		piLines, ompigLines := strings.Split(string(a), "\n"), strings.Split(string(b), "\n")
		if len(piLines) != len(ompigLines) {
			return differing + 1
		}
		if entry.Name() != "main.jsonl" {
			continue // sidecar files list document and task members in construction order, which the generator does not reproduce
		}
		for k := range piLines {
			if strings.Contains(piLines[k], `"type":"task"`) || strings.Contains(piLines[k], `"type":"submission"`) || strings.Contains(piLines[k], `"payload":{"type":"task"`) {
				continue
			}
			if piLines[k] != ompigLines[k] {
				if differing++; reported+differing <= 8 {
					t.Errorf("program %d %s line %d\n  Pi  %.400s\n  Ompig %.400s", program, entry.Name(), k, piLines[k], ompigLines[k])
				}
			}
		}
	}
	return differing
}
