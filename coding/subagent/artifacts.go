package subagent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Artifacts owns the directory that holds a session's subagent outputs.
//
// Upstream: omp src/task/output-manager.ts (AgentOutputManager). The layout is
// flat under the owning session's artifact directory:
//
//	<id>.md       final output, what `agent://<id>` resolves to
//	<id>.json     structured output sidecar, when the run produced data
//	<id>.jsonl    the child session transcript
//	<id>.patch    captured isolation patch, when isolation produced one
type Artifacts struct {
	dir string
}

// NewArtifacts returns the store rooted at dir, creating it on first write.
func NewArtifacts(dir string) *Artifacts { return &Artifacts{dir: dir} }

// Dir reports the store root.
func (a *Artifacts) Dir() string {
	if a == nil {
		return ""
	}
	return a.dir
}

// OutputPath is where an agent's final output lives.
func (a *Artifacts) OutputPath(id string) string {
	if a == nil || a.dir == "" || id == "" {
		return ""
	}
	return filepath.Join(a.dir, sanitizeFile(id)+".md")
}

// StructuredPath is where an agent's structured output lives.
func (a *Artifacts) StructuredPath(id string) string {
	if a == nil || a.dir == "" || id == "" {
		return ""
	}
	return filepath.Join(a.dir, sanitizeFile(id)+".json")
}

// TranscriptPath is where an agent's session transcript lives.
func (a *Artifacts) TranscriptPath(id string) string {
	if a == nil || a.dir == "" || id == "" {
		return ""
	}
	return filepath.Join(a.dir, sanitizeFile(id)+".jsonl")
}

// WriteTranscriptNote records where an agent's transcript actually lives. The
// session manager names a child session file itself, so the path cannot be
// derived from the agent id after a restart.
func (a *Artifacts) WriteTranscriptNote(id, transcriptPath string) error {
	if a == nil || a.dir == "" || id == "" || transcriptPath == "" {
		return nil
	}
	if err := os.MkdirAll(a.dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.dir, sanitizeFile(id)+".session"), []byte(transcriptPath), 0o644)
}

// ReadTranscriptNote returns the recorded transcript path, when one exists.
func (a *Artifacts) ReadTranscriptNote(id string) (string, bool) {
	if a == nil || a.dir == "" || id == "" {
		return "", false
	}
	content, err := os.ReadFile(filepath.Join(a.dir, sanitizeFile(id)+".session"))
	if err != nil {
		return "", false
	}
	path := strings.TrimSpace(string(content))
	if path == "" {
		return "", false
	}
	return path, true
}

// WriteOutput stores an agent's final output. A structured payload, when
// non-nil, is written beside it. It returns the output path.
func (a *Artifacts) WriteOutput(id, text string, structured json.RawMessage) (string, error) {
	if a == nil || a.dir == "" || id == "" {
		return "", fmt.Errorf("subagent: no artifacts directory")
	}
	if err := os.MkdirAll(a.dir, 0o755); err != nil {
		return "", fmt.Errorf("subagent: create artifacts dir: %w", err)
	}
	path := a.OutputPath(id)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return "", fmt.Errorf("subagent: write output: %w", err)
	}
	structuredPath := a.StructuredPath(id)
	if len(structured) > 0 {
		if err := os.WriteFile(structuredPath, structured, 0o644); err != nil {
			return path, fmt.Errorf("subagent: write structured output: %w", err)
		}
	} else {
		// A stale sidecar would make a later JSON read return an old payload.
		_ = os.Remove(structuredPath)
	}
	return path, nil
}

// ReadOutput returns an agent's saved final output.
func (a *Artifacts) ReadOutput(id string) (string, error) {
	path := a.OutputPath(id)
	if path == "" {
		return "", fmt.Errorf("subagent: no artifacts directory")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("subagent: read %s: %w", id, err)
	}
	return string(content), nil
}

// ReadStructured returns an agent's structured sidecar, preferring it over
// parsing the Markdown output, as upstream's JSON-path reads do.
func (a *Artifacts) ReadStructured(id string) (json.RawMessage, bool, error) {
	path := a.StructuredPath(id)
	if path == "" {
		return nil, false, fmt.Errorf("subagent: no artifacts directory")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return json.RawMessage(content), true, nil
}

// JSONPath extracts one value from an agent's structured output by path
// segments, e.g. "reports/0/data". It reports whether a value was found.
func (a *Artifacts) JSONPath(id string, segments []string) (json.RawMessage, bool, error) {
	raw, ok, err := a.ReadStructured(id)
	if err != nil || !ok {
		return nil, false, err
	}
	current := json.RawMessage(raw)
	for _, segment := range segments {
		// An object step decodes into a map of raw values, so the tree is walked
		// without decoding anything into an interface value.
		var object map[string]json.RawMessage
		if err := json.Unmarshal(current, &object); err == nil {
			next, exists := object[segment]
			if !exists {
				return nil, false, nil
			}
			current = next
			continue
		}
		var array []json.RawMessage
		if err := json.Unmarshal(current, &array); err != nil {
			return nil, false, nil
		}
		index, convErr := strconv.Atoi(segment)
		if convErr != nil || index < 0 || index >= len(array) {
			return nil, false, nil
		}
		current = array[index]
	}
	return current, true, nil
}

// List returns the ids with a stored output, sorted.
func (a *Artifacts) List() []string {
	if a == nil || a.dir == "" {
		return nil
	}
	entries, err := os.ReadDir(a.dir)
	if err != nil {
		return nil
	}
	var ids []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".md"))
	}
	sort.Strings(ids)
	return ids
}

// sanitizeFile keeps an agent id usable as one file name: nesting is
// dot-qualified, which is fine, but separators and parent references are not.
func sanitizeFile(id string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", "..", "-")
	out := replacer.Replace(strings.TrimSpace(id))
	if out == "" {
		return "agent"
	}
	return out
}

// Record is one agent's on-disk footprint, read back after a restart.
type Record struct {
	ID             string
	OutputPath     string
	StructuredPath string
	TranscriptPath string
	// Size is the output's byte count, which is also its token proxy.
	Size int64
}

// Scan returns every agent with a stored transcript or output, sorted by id.
// It is how the Agent Hub repopulates its roster when a session resumes: the
// registry is process state, the transcripts are not.
func (a *Artifacts) Scan() []Record {
	if a == nil || a.dir == "" {
		return nil
	}
	entries, err := os.ReadDir(a.dir)
	if err != nil {
		return nil
	}
	byID := map[string]*Record{}
	record := func(id string) *Record {
		if existing, ok := byID[id]; ok {
			return existing
		}
		created := &Record{ID: id}
		byID[id] = created
		return created
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		base, ext := splitArtifact(name)
		if base == "" {
			continue
		}
		switch ext {
		case ".jsonl":
			// The child's own session file is written by the session manager with
			// a timestamped name; it is not an agent artifact, so it is skipped.
			if looksLikeSessionFile(base) {
				continue
			}
			record(base).TranscriptPath = filepath.Join(a.dir, name)
		case ".md":
			rec := record(base)
			rec.OutputPath = filepath.Join(a.dir, name)
			if info, err := entry.Info(); err == nil {
				rec.Size = info.Size()
			}
		case ".json":
			record(base).StructuredPath = filepath.Join(a.dir, name)
		case ".session":
			if note, ok := a.ReadTranscriptNote(base); ok {
				record(base).TranscriptPath = note
			}
		}
	}
	out := make([]Record, 0, len(byID))
	for _, rec := range byID {
		out = append(out, *rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// looksLikeSessionFile reports whether a base name is a timestamped session
// file ("2026-10-29T08-49-11-413Z_<uuid>") rather than an agent id.
func looksLikeSessionFile(base string) bool {
	return len(base) > 16 && base[4] == '-' && base[10] == 'T' && strings.Contains(base, "_")
}

// splitArtifact splits "Parent.Child.md" into its id and extension.
func splitArtifact(name string) (string, string) {
	switch {
	// .jsonl first: ".json" is a prefix of ".jsonl", so testing it first would
	// read a transcript as "a.jsonl" plus a .json extension.
	case strings.HasSuffix(name, ".jsonl"):
		return strings.TrimSuffix(name, ".jsonl"), ".jsonl"
	case strings.HasSuffix(name, ".json"):
		return strings.TrimSuffix(name, ".json"), ".json"
	case strings.HasSuffix(name, ".md"):
		return strings.TrimSuffix(name, ".md"), ".md"
	case strings.HasSuffix(name, ".session"):
		return strings.TrimSuffix(name, ".session"), ".session"
	}
	return "", ""
}
