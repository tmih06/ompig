package cli

import (
	"slices"

	ompiglet "github.com/tmih06/ompig/coding/ompiglet"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// applyPigletStrip records the active Ompiglet's strip list in pigstrip, the
// process's one strip state, and sets the Pi switches the stripped tools and
// features reuse from that state: a stripped tool joins the --exclude-tools
// denylist, so it leaves the tool registry, the active tools and the system
// prompt; a stripped skills, prompt-templates or themes feature sets Pi's
// --no-skills, --no-prompt-templates or --no-themes switch and drops the
// explicit paths. Every other registry asks pigstrip where it registers a
// built-in, so a runtime-stripped built-in and one a Ompiglet Binary compiled
// out are the same record. The returned undo removes the records this call
// added; the process keeps them for its lifetime.
// ompig additive (D92): a Ompiglet strip list disables built-ins at runtime.
func applyPigletStrip(p *ompiglet.Ompiglet, flags *Args) (undo func()) {
	if p == nil || p.Strip.IsEmpty() {
		return func() {}
	}
	undoName := pigstrip.SetPiglet(p.Name)
	undoRecord := p.Strip.Record()
	undo = func() { undoRecord(); undoName() }
	for _, name := range pigstrip.IDs(pigstrip.ListTools) {
		if !slices.Contains(flags.ExcludeTools, name) {
			flags.ExcludeTools = append(flags.ExcludeTools, name)
		}
	}
	if pigstrip.Has(pigstrip.ListFeatures, pigstrip.Skills) {
		flags.NoSkills = true
		flags.Skills = nil
	}
	if pigstrip.Has(pigstrip.ListFeatures, pigstrip.PromptTemplates) {
		flags.NoPromptTemplates = true
		flags.PromptTemplates = nil
	}
	if pigstrip.Has(pigstrip.ListFeatures, pigstrip.Themes) {
		flags.NoThemes = true
		flags.Themes = nil
	}
	return undo
}
