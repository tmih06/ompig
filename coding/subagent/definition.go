package subagent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tmih06/ompig/internal/codingagent/frontmatter"
)

// Definition is one discoverable agent: a Markdown file with YAML frontmatter
// and a system-prompt body.
//
// Upstream: omp src/task/types.ts AgentDefinition.
type Definition struct {
	Name         string
	Description  string
	SystemPrompt string
	// Tools selects the child's built-ins. Nil means "the default set"; a
	// non-nil empty slice means "no built-in tools".
	Tools []string
	// Spawns limits the agent types this agent may spawn: "*", a CSV/array of
	// names, or nil to inherit the parent's policy.
	Spawns []string
	// Model is a prioritized list of selectors; the first resolvable wins.
	Model         []string
	ThinkingLevel string
	// Blocking runs the agent inline even when background execution is on.
	Blocking bool
	// Source is where the definition came from: bundled, user, project, plugin.
	Source string
	// FilePath is the Markdown file, empty for bundled definitions.
	FilePath string
}

// Discovery captures the directories an agent search reads.
type Discovery struct {
	// ProjectDir is <cwd>/.ompig/agents.
	ProjectDir string
	// UserDir is <agentDir>/agents.
	UserDir string
	// PluginDirs are extension-package agents/ roots, in precedence order.
	PluginDirs []string
}

// reservedAgentNames are rejected by upstream: they would shadow the main and
// subagent pseudo-agents.
var reservedAgentNames = map[string]struct{}{"main": {}, "sub": {}}

// Load discovers agent definitions in first-wins precedence:
// project, user, then plugin roots. Bundled definitions are appended by
// BundledDefinitions and filtered against the discovered set by the caller.
func Load(d Discovery) ([]Definition, []error) {
	var (
		out  []Definition
		errs []error
	)
	seen := map[string]struct{}{}
	add := func(def Definition) {
		if _, dup := seen[def.Name]; dup {
			return
		}
		seen[def.Name] = struct{}{}
		out = append(out, def)
	}

	dirs := []struct {
		path   string
		source string
	}{
		{d.ProjectDir, "project"},
		{d.UserDir, "user"},
	}
	for _, plugin := range d.PluginDirs {
		dirs = append(dirs, struct {
			path   string
			source string
		}{plugin, "plugin"})
	}

	for _, dir := range dirs {
		if dir.path == "" {
			continue
		}
		defs, dirErrs := loadDir(dir.path, dir.source)
		errs = append(errs, dirErrs...)
		for _, def := range defs {
			add(def)
		}
	}
	return out, errs
}

// loadDir parses every .md file in dir in lexicographic order. One bad file is
// a reported error and does not stop the scan, matching upstream.
func loadDir(dir, source string) ([]Definition, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []error{fmt.Errorf("read agents dir %s: %w", dir, err)}
	}
	var (
		defs []Definition
		errs []error
	)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("read agent %s: %w", path, err))
			continue
		}
		def, err := ParseDefinition(string(content))
		if err != nil {
			errs = append(errs, fmt.Errorf("parse agent %s: %w", path, err))
			continue
		}
		def.Source = source
		def.FilePath = path
		defs = append(defs, def)
	}
	return defs, errs
}

// ParseDefinition parses one agent Markdown file. A definition without a name
// or description is invalid, as upstream's parseAgentFields rejects it.
func ParseDefinition(content string) (Definition, error) {
	doc := frontmatter.Parse(content)
	if doc.Err != nil {
		return Definition{}, doc.Err
	}
	fields := doc.Frontmatter
	name, _ := fields["name"].(string)
	name = strings.TrimSpace(name)
	description, _ := fields["description"].(string)
	if name == "" {
		return Definition{}, fmt.Errorf("missing name")
	}
	if description == "" {
		return Definition{}, fmt.Errorf("missing description")
	}
	if _, reserved := reservedAgentNames[strings.ToLower(name)]; reserved {
		return Definition{}, fmt.Errorf("agent name %q is reserved", name)
	}
	def := Definition{
		Name:         name,
		Description:  description,
		SystemPrompt: strings.TrimSpace(doc.Body),
		Tools:        stringList(fields["tools"], true),
		Spawns:       stringList(fields["spawns"], false),
		Model:        stringList(fields["model"], false),
	}
	if level, ok := fields["thinking-level"].(string); ok {
		def.ThinkingLevel = strings.TrimSpace(level)
	} else if level, ok := fields["thinking"].(string); ok {
		def.ThinkingLevel = strings.TrimSpace(level)
	}
	def.Blocking = truthy(fields["blocking"])
	return def, nil
}

// stringList reads a frontmatter value that may be a scalar, a CSV string or a
// list. nilInput reports the distinction between an absent key and an explicit
// empty one: when absent, nil is returned.
func stringList(value any, nilWhenAbsent bool) []string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case string:
		parts := strings.Split(typed, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				if text = strings.TrimSpace(text); text != "" {
					out = append(out, text)
				}
			}
		}
		return out
	default:
		_ = nilWhenAbsent
		return nil
	}
}

// truthy reads a YAML-ish boolean.
func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "on":
			return true
		}
	}
	return false
}

// CanSpawn reports whether the definition may spawn subagents. An omitted
// `spawns` inherits the parent's policy (allowed); an explicit empty value
// denies spawning; a list allows only the named agents; `*` allows any.
func (d Definition) CanSpawn() bool {
	if d.Spawns == nil {
		return true
	}
	return len(d.Spawns) > 0
}

// MaySpawn reports whether this definition may spawn the named agent.
func (d Definition) MaySpawn(name string) bool {
	if d.Spawns == nil {
		return true
	}
	for _, allowed := range d.Spawns {
		if allowed == "*" || strings.EqualFold(allowed, name) {
			return true
		}
	}
	return false
}

// CanSpawnAgent reports whether the runtime may spawn name on behalf of the
// agent that runs def. A depth-limited or denied spawn is rejected before a
// child session exists.
func (d Definition) CanSpawnAgent(name string) bool {
	if strings.TrimSpace(name) == "" {
		name = "task"
	}
	return d.MaySpawn(name)
}
