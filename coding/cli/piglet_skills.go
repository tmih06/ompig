package cli

import (
	"errors"
	"fmt"

	"github.com/tmih06/ompig/coding/extension"
	ompiglet "github.com/tmih06/ompig/coding/ompiglet"
	"github.com/tmih06/ompig/internal/codingagent"
)

// skillLoadResult holds the outcome of resolving and loading all skills for a
// session: from the ompiglet (if any), --skill flags, and convention dirs.
type skillLoadResult struct {
	// Defs are the fully-loaded skill definitions, ready for the system prompt
	// and /skill: command registration.
	Defs []*codingagent.SkillDef
	// Paths are the source paths from which Defs were loaded (for the
	// SkillPaths interactive-mode option and hot-reload).
	Paths       []string
	Diagnostics []extension.ResourceDiagnostic
}

// resolveAndLoadSkills resolves and loads every skill for the session.
//
// When a ompiglet declares skills, they are resolved and loaded *in addition
// to* (not instead of) the marketplace/convention skills that
// collectSkillInputs already gathered. This is the union semantics documented
// in the adwaraki ompiglet:
//
//	"Marketplace skills ... are inherited globally via settings.json packages:,
//	 so they load for any ompiglet ... without being redeclared here."
//
// If a ompiglet skill has the same name as a marketplace skill, the ompiglet
// version wins (it is the explicit declaration). Deduplication is by name,
// not by path: the two sources have different paths for the same skill.
//
// Extracted from main() so the ompiglet-skills → loadSkills wiring: the site
// of a bug where NoSkills was passed as loadSkills's kill-switch: is unit
// testable. The loadSkills call inside this function always passes false for
// noSkills: we always want to load explicitly-resolved inputs.
func resolveAndLoadSkills(p *ompiglet.Ompiglet, collectedSkillInputs []string) (*skillLoadResult, error) {
	if p == nil || len(p.Skills) == 0 {
		// No ompiglet skills: just load whatever collectSkillInputs gathered.
		defs, diagnostics, err := loadSkills(collectedSkillInputs, false)
		return &skillLoadResult{Defs: defs, Paths: collectedSkillInputs, Diagnostics: diagnostics}, err
	}

	// Ompiglet declares skills: resolve origins and build the ompiglet skill
	// inputs (filesystem paths + inline content).
	resolved, errs := ompiglet.ResolveSkills(p)
	if len(errs) > 0 {
		return nil, fmt.Errorf("resolve Ompiglet skills: %w", errors.Join(errs...))
	}

	var pigletSkillInputs []string
	var inlineDefs []*codingagent.SkillDef
	for _, s := range p.Skills {
		if s.Content == "" {
			continue
		}
		inlineDefs = append(inlineDefs, &codingagent.SkillDef{
			Name:        s.Name,
			Description: s.Description,
			Body:        s.Content,
		})
	}
	for _, s := range resolved {
		pigletSkillInputs = append(pigletSkillInputs, s.Path)
	}

	// Union: load marketplace/convention skills first, then ompiglet skills on
	// top. Deduplicate by NAME (not path) so the ompiglet version wins for
	// overlaps. Load each set separately so ambient first-wins collision
	// handling does not suppress an explicit Ompiglet selection.
	marketDefs, diagnostics, err := loadSkills(collectedSkillInputs, false)
	if err != nil {
		return nil, err
	}
	pigletDefs, pigletDiagnostics, err := loadSkills(pigletSkillInputs, false)
	if err != nil {
		return nil, err
	}

	// Build a name→def map. Start with marketplace skills, then overlay
	// ompiglet skills (ompiglet wins on name conflict).
	byName := make(map[string]*codingagent.SkillDef, len(marketDefs)+len(pigletDefs))
	var orderedNames []string
	for _, d := range marketDefs {
		if _, exists := byName[d.Name]; !exists {
			byName[d.Name] = d
			orderedNames = append(orderedNames, d.Name)
		}
	}
	for _, d := range pigletDefs {
		if _, exists := byName[d.Name]; !exists {
			// New ompiglet-only skill: add it (keep stable ordering).
			byName[d.Name] = d
			orderedNames = append(orderedNames, d.Name)
		} else {
			// Name conflict: ompiglet version wins (replace).
			byName[d.Name] = d
		}
	}
	for _, d := range inlineDefs {
		if _, exists := byName[d.Name]; !exists {
			orderedNames = append(orderedNames, d.Name)
		}
		byName[d.Name] = d
	}

	// Assemble the final deduplicated list in stable order.
	var defs []*codingagent.SkillDef
	for _, name := range orderedNames {
		defs = append(defs, byName[name])
	}

	// SkillPaths: union of marketplace + ompiglet paths, for /reload.
	allPaths := dedupStrings(append(append([]string{}, pigletSkillInputs...), collectedSkillInputs...))

	return &skillLoadResult{Defs: defs, Paths: allPaths, Diagnostics: append(diagnostics, pigletDiagnostics...)}, nil
}
