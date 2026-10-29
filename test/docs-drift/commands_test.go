package docsdrift

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// The documented `ompig` command lines are checked against the real binary: the subcommand must be one its help lists, every flag must appear in the help of the command it is given to, and the read-only commands run and exit 0. The expected set comes from the binary's own help, so the gate cannot drift from the thing it guards.

var (
	ompigOnce sync.Once
	ompigPath string
	ompigErr  error
	ompigHome string
)

// ompigBinary builds cmd/ompig once for the package's checks.
func ompigBinary(t *testing.T) string {
	t.Helper()
	ompigOnce.Do(func() {
		dir, err := os.MkdirTemp("", "docs-drift-ompig-")
		if err != nil {
			ompigErr = err
			return
		}
		ompigHome = filepath.Join(dir, "home")
		if err = os.MkdirAll(ompigHome, 0o755); err != nil {
			ompigErr = err
			return
		}
		ompigPath = filepath.Join(dir, "ompig")
		if runtime.GOOS == "windows" {
			ompigPath += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", ompigPath, "./cmd/ompig")
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			ompigErr = fmt.Errorf("go build ./cmd/ompig: %w\n%s", err, out)
		}
	})
	if ompigErr != nil {
		t.Fatal(ompigErr)
	}
	return ompigPath
}

func TestMain(m *testing.M) {
	code := m.Run()
	if ompigPath != "" {
		_ = os.RemoveAll(filepath.Dir(ompigPath))
	}
	os.Exit(code)
}

// ompigEnv isolates every run from the developer's own configuration and credentials.
func ompigEnv() []string {
	env := []string{}
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "OMPIG_") || strings.HasPrefix(name, "PI_") || name == "HOME" || name == "USERPROFILE" {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"HOME="+ompigHome, "USERPROFILE="+ompigHome,
		"OMPIG_HOME="+filepath.Join(ompigHome, ".ompig"), "PI_HOME="+filepath.Join(ompigHome, ".pi"),
		"OMPIG_CODING_AGENT_DIR="+filepath.Join(ompigHome, "agent"), "PI_CODING_AGENT_DIR="+filepath.Join(ompigHome, "pi-agent"),
		"OMPIG_OFFLINE=1", "PI_OFFLINE=1", "NO_COLOR=1", "FORCE_COLOR=0")
}

var (
	helpMu    sync.Mutex
	helpCache = map[string]string{}
)

// runPig runs the binary with a timeout and returns its combined output and exit code.
func runPig(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(ompigBinary(t), args...)
	cmd.Env = ompigEnv()
	cmd.Dir = ompigHome
	done := make(chan struct{})
	var out []byte
	var err error
	go func() {
		out, err = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("ompig %v did not finish in 60s", args)
	}
	code := 0
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("ompig %v: %v", args, err)
	}
	return string(out), code
}

// helpOf returns the help text of a command path, empty when the path has none.
func helpOf(t *testing.T, path ...string) string {
	t.Helper()
	key := strings.Join(path, " ")
	helpMu.Lock()
	defer helpMu.Unlock()
	if h, ok := helpCache[key]; ok {
		return h
	}
	out, _ := runPig(t, append(append([]string{}, path...), "--help")...)
	helpCache[key] = out
	return out
}

var (
	flagToken  = regexp.MustCompile(`(?:^|[\s,\[(|])(--?[A-Za-z][A-Za-z0-9-]*)`)
	spanPig    = regexp.MustCompile("`(ompig(?: [^`\\n]+)?)`")
	shellSplit = regexp.MustCompile(`'[^']*'|"(?:[^"\\]|\\.)*"|\S+`)
)

// helpFlags returns every flag a help text names.
func helpFlags(help string) map[string]bool {
	out := map[string]bool{}
	for _, m := range flagToken.FindAllStringSubmatch(help, -1) {
		out[m[1]] = true
	}
	return out
}

// extensionFlags are flags that extensions register, which the core help names only as an example.
var extensionFlags = map[string]string{
	"--plan":  "the plan-mode example extension (ompig --help: Extensions can register additional flags)",
	"--shout": "the tour extension of first-extension.md, which registers it with e.Flag",
	"--ompig": "the ompig-acp companion executable of the Pigpen acp Package (packages.md), not a ompig flag",
}

// commandWords splits a documented command line the way a shell does, up to the first control operator.
func commandWords(line string) []string {
	var words []string
	for _, w := range shellSplit.FindAllString(line, -1) {
		switch w {
		case "|", "||", "&&", ";", ">", ">>", "<", "2>&1", "&":
			return words
		}
		if len(w) >= 2 && (w[0] == '"' || w[0] == '\'') && w[len(w)-1] == w[0] {
			w = w[1 : len(w)-1]
		}
		words = append(words, w)
	}
	return words
}

var (
	wordRE     = regexp.MustCompile(`^[a-z][a-z-]*$`)
	flagWordRE = regexp.MustCompile(`^--?[A-Za-z][A-Za-z0-9-]*$`)
)

// isMainHelp reports whether out is the top-level help, which the binary prints for a word it does not know.
func isMainHelp(out string) bool { return strings.HasPrefix(out, "ompig - AI coding assistant") }

// isCommandWord reports whether w is a command or subcommand under path: the parent's help lists `ompig <path> <w>`, or `ompig <path> <w> --help` prints help of its own.
func isCommandWord(t *testing.T, path []string, w, parentHelp string) bool {
	t.Helper()
	next := append(append([]string{}, path...), w)
	if regexp.MustCompile(`(?m)\bpig ` + regexp.QuoteMeta(strings.Join(next, " ")) + `\b`).MatchString(parentHelp) {
		return true
	}
	own := helpOf(t, next...)
	return own != "" && !isMainHelp(own) && own != parentHelp
}

// validatePigCommand checks one documented invocation and returns the problems it finds. The flags a command accepts are the flags of its own help, of its ancestors' help (a group's usage lists the flags of each form), and of the help of the form the line selects with --to.
func validatePigCommand(t *testing.T, words []string) []string {
	t.Helper()
	if len(words) == 0 || words[0] != "ompig" {
		return nil
	}
	args := words[1:]
	var problems []string
	path := []string{}
	help := helpOf(t)
	known := helpFlags(help)
	i := 0
	for ; i < len(args) && wordRE.MatchString(args[i]); i++ {
		if !isCommandWord(t, path, args[i], help) {
			break
		}
		path = append(path, args[i])
		if sub := helpOf(t, path...); sub != "" && !isMainHelp(sub) {
			help = sub
			for f := range helpFlags(sub) {
				known[f] = true
			}
		}
	}
	for j := i; j < len(args)-1; j++ {
		if args[j] == "--to" {
			for f := range helpFlags(helpOf(t, append(append([]string{}, path...), "--to", args[j+1])...)) {
				known[f] = true
			}
		}
	}
	for _, w := range args[i:] {
		if w == "--" {
			break
		}
		if !strings.HasPrefix(w, "-") || w == "-" {
			continue
		}
		flag, _, _ := strings.Cut(w, "=")
		if !flagWordRE.MatchString(flag) || known[flag] || extensionFlags[flag] != "" || flag == "--help" || flag == "-h" {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s is not a flag of `ompig %s`", flag, strings.Join(path, " ")))
	}
	if len(path) == 0 && len(args) == 1 && wordRE.MatchString(args[0]) && !promptWords[args[0]] {
		problems = append(problems, fmt.Sprintf("`ompig %s` is not a command", args[0]))
	}
	return problems
}

// promptWords are single words the docs pass as a prompt, not as a command.
var promptWords = map[string]bool{}

var safeToRun = map[string]bool{
	"--version": true, "-v": true, "--help": true, "-h": true,
	"version": true, "status": true,
	"docs list": true, "docs path": true, "setup status": true, "ompiglet schema": true, "ompiglet list": true,
}

func TestDocumentedPigCommandsAreValid(t *testing.T) {
	ompigBinary(t)
	checked := map[string]bool{}
	ran := 0
	for _, src := range corpus(t) {
		for _, line := range ompigCommandLines(src) {
			words := commandWords(line.Text)
			problems := validatePigCommand(t, words)
			for _, p := range problems {
				t.Errorf("%s:%d: %q: %s", src.Rel, line.Line, line.Text, p)
			}
			key := strings.Join(words[1:], " ")
			if len(problems) == 0 && safeToRun[key] && !checked[key] {
				checked[key] = true
				ran++
				if out, code := runPig(t, words[1:]...); code != 0 {
					t.Errorf("%s:%d: `%s` exits %d:\n%s", src.Rel, line.Line, line.Text, code, out)
				}
			}
		}
	}
	if ran == 0 {
		t.Fatal("no documented command was run; the safe list matches nothing")
	}
}

type commandLine struct {
	Line int
	Text string
}

// ompigCommandLines returns the command lines that start with `ompig`: the lines of shell samples and the inline code spans of prose.
func ompigCommandLines(src docSource) []commandLine {
	var out []commandLine
	inFence, shellFence := false, false
	var fence string
	lines := strings.Split(src.Body, "\n")
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if m := fenceOpen.FindStringSubmatch(l); m != nil && !inFence {
			inFence, fence = true, m[2]
			switch strings.ToLower(m[3]) {
			case "bash", "sh", "shell", "console":
				shellFence = !skipBefore(lines, i)
			default:
				shellFence = false
			}
			continue
		}
		if inFence {
			if closesFence(l, fence) {
				inFence, shellFence = false, false
				continue
			}
			if !shellFence {
				continue
			}
			t := strings.TrimSpace(l)
			t = strings.TrimPrefix(t, "$ ")
			for strings.HasSuffix(t, "\\") && i+1 < len(lines) {
				i++
				t = strings.TrimSuffix(t, "\\") + " " + strings.TrimSpace(lines[i])
			}
			if t == "ompig" || strings.HasPrefix(t, "ompig ") {
				out = append(out, commandLine{i + 1, t})
			}
			continue
		}
		for _, m := range spanPig.FindAllStringSubmatch(l, -1) {
			out = append(out, commandLine{i + 1, m[1]})
		}
	}
	return out
}

func skipBefore(lines []string, i int) bool {
	return i > 0 && skipMark.MatchString(strings.TrimSpace(lines[i-1]))
}

func TestPigHelpIsStableUnderTheDocsIsolation(t *testing.T) {
	out, code := runPig(t, "--help")
	if code != 0 || !strings.Contains(out, "Usage:") {
		t.Fatalf("ompig --help exits %d:\n%s", code, out)
	}
	flags := helpFlags(out)
	var names []string
	for f := range flags {
		names = append(names, f)
	}
	sort.Strings(names)
	if len(names) < 30 {
		t.Fatalf("the help parser found only %d flags: %v", len(names), names)
	}
}

func TestPigCommandValidatorRejectsWhatDoesNotExist(t *testing.T) {
	for _, c := range []struct {
		line string
		bad  bool
	}{
		{"ompig --print --model sonnet:high \"hello\"", false},
		{"ompig install npm:@foo/bar -l", false},
		{"ompig ompiglet publish ./x.yaml --to npm --yes --access public", false},
		{"ompig mcp add fs -- npx -y server", false},
		{"ompig --no-such-flag \"hello\"", true},
		{"ompig install --bogus ./x", true},
		{"ompig frobnicate", true},
		{"ompig ompiglet build x --nonsense", true},
	} {
		got := validatePigCommand(t, commandWords(c.line))
		if (len(got) > 0) != c.bad {
			t.Errorf("validate(%q) = %v, want bad=%v", c.line, got, c.bad)
		}
	}
}

func TestCommandWordsStopAtShellOperators(t *testing.T) {
	got := commandWords(`ompig -p "a b" --model x | tee out.txt`)
	want := []string{"ompig", "-p", "a b", "--model", "x"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("commandWords = %q, want %q", got, want)
	}
}

// Every flag the docs name in backticks must be a flag of some Ompig command, unless the page says it belongs to another tool.
var foreignFlags = map[string]string{
	"--yes": "npm publish and others", "--dry-run": "npm and git examples", "--registry": "npm", "--save-dev": "npm",
	"--arm": "k8sbench (the benchmark harness in benchmarking.md)", "--expose-gc": "Node (the benchmark notes in benchmarking.md)", "--rm": "docker", "--version": "any tool", "--help": "any tool", "--tag": "npm publish", "--access": "npm publish", "--otp": "npm publish",
	"--omit": "npm install", "--no-optional": "npm install", "--ignore-scripts": "npm install (the flag Ompig passes while it resolves a Ompiglet, ompiglets.md)",
	"--models-dir": "llama-server", "--no-models-autoload": "llama-server", "--jinja": "llama-server", "--host": "llama-server", "--reasoning-budget": "llama-server",
	"--models-preset": "llama-server (nvidia-rtx-llama-cpp.md)", "--ctx-size": "llama-server (nvidia-rtx-llama-cpp.md)", "--port": "llama-server (nvidia-rtx-llama-cpp.md)", "--spec-draft-model": "llama-server (nvidia-rtx-llama-cpp.md)",
	"--ctx-checkpoints": "llama-server (llama-cpp.md)", "--checkpoint-min-step": "llama-server (llama-cpp.md)",
	"--json-only": "the model catalog generator (development.md)", "--json-output": "the model catalog generator (development.md)", "--strict": "the model catalog generator (development.md)",
	// A hidden flag: `ompig --help` omits it on purpose, because it is an explicit unsafe opt-in. Ompiglet runtime (coding/ompiglet/runtime.go) reads it.
	"--unsafe-host":                "hidden unsafe opt-in of the Ompiglet runtime",
	"--no-install":                 "Bun, as Ompig starts it for the TypeScript extension runtime (typescript-runtimes.md)",
	"--config":                     "Bun, as Ompig starts it for the TypeScript extension runtime (typescript-runtimes.md)",
	"--no-env-file":                "Bun, as Ompig starts it for the TypeScript extension runtime (typescript-runtimes.md)",
	"--no-compile-autoload-dotenv": "bun build, for a compiled TypeScript runtime (typescript-runtimes.md)",
	"--no-compile-autoload-bunfig": "bun build, for a compiled TypeScript runtime (typescript-runtimes.md)",
}

var backtickFlag = regexp.MustCompile("`(--[a-z][a-z0-9-]*)(?:[ =][^`]*)?`")

func allPigFlags(t *testing.T) map[string]bool {
	t.Helper()
	all := helpFlags(helpOf(t))
	var walk func(path []string, depth int)
	words := regexp.MustCompile(`(?m)\bompig((?: [a-z][a-z-]*)+)`)
	seen := map[string]bool{}
	walk = func(path []string, depth int) {
		help := helpOf(t, path...)
		for f := range helpFlags(help) {
			all[f] = true
		}
		if depth >= 3 {
			return
		}
		for _, m := range words.FindAllStringSubmatch(help, -1) {
			next := strings.Fields(m[1])
			key := strings.Join(next, " ")
			if len(next) <= len(path) || seen[key] || strings.Join(next[:len(path)], " ") != strings.Join(path, " ") {
				continue
			}
			seen[key] = true
			walk(next[:len(path)+1], depth+1)
		}
	}
	for _, m := range words.FindAllStringSubmatch(helpOf(t), -1) {
		next := strings.Fields(m[1])
		if !seen[next[0]] {
			seen[next[0]] = true
			walk(next[:1], 1)
		}
	}
	for _, extra := range [][]string{{"login"}, {"logout"}, {"reload"}, {"package"}, {"extensions"}, {"ompiglet", "publish", "--to", "npm"}, {"package", "publish", "--to", "npm"}} {
		walk(extra, 3)
	}
	return all
}

func TestDocumentedFlagsExist(t *testing.T) {
	flags := allPigFlags(t)
	for _, src := range corpus(t) {
		if strings.HasSuffix(src.Rel, "bug-ledger.md") {
			continue
		}
		for i, line := range strings.Split(src.Body, "\n") {
			for _, m := range backtickFlag.FindAllStringSubmatch(line, -1) {
				f := m[1]
				if flags[f] || foreignFlags[f] != "" || extensionFlags[f] != "" {
					continue
				}
				t.Errorf("%s:%d: documents %s, which no ompig command's help lists", src.Rel, i+1, f)
			}
		}
	}
}

// A flag `ompig --help` lists must be in the CLI reference, because `--help` is what a user trusts and the reference is where they look next. The set comes from the binary.
func TestEveryHelpOptionIsInTheCLIReference(t *testing.T) {
	help := helpOf(t)
	_, options, ok := strings.Cut(help, "\nOptions:\n")
	if !ok {
		t.Fatal("ompig --help has no Options section")
	}
	options, _, _ = strings.Cut(options, "\nExtensions can register")
	var flags []string
	for _, m := range regexp.MustCompile(`(?m)^\s{2}(--[a-z][a-z0-9-]*)`).FindAllStringSubmatch(options+"\nOmpig Options:\n"+ompigOptions(help), -1) {
		flags = append(flags, m[1])
	}
	if len(flags) < 30 {
		t.Fatalf("found %d options in ompig --help; the parser is broken", len(flags))
	}
	var pages []docSource
	for _, src := range corpus(t) {
		if src.Rel == "docs/site/docs/cli.md" || src.Rel == "internal/pigdocs/content/commands.md" {
			pages = append(pages, src)
		}
	}
	if len(pages) != 2 {
		t.Fatalf("the CLI reference pages are missing from the corpus: %d found", len(pages))
	}
	for _, page := range pages {
		for _, f := range flags {
			if !strings.Contains(page.Body, f) {
				t.Errorf("%s does not document %s, which `ompig --help` lists", page.Rel, f)
			}
		}
	}
}

func ompigOptions(help string) string {
	_, rest, _ := strings.Cut(help, "\nOmpig Options:\n")
	rest, _, _ = strings.Cut(rest, "\n\n")
	return rest
}

var helpEnvLine = regexp.MustCompile(`(?m)^  ([A-Z][A-Z0-9_]+)\s+-`)

// The environment variables `ompig --help` lists are the ones Ompig reads. Each must be in some page of the corpus.
func TestEveryHelpEnvironmentVariableIsDocumented(t *testing.T) {
	help := helpOf(t)
	_, section, ok := strings.Cut(help, "Environment Variables:")
	if !ok {
		t.Fatal("ompig --help has no Environment Variables section")
	}
	var all strings.Builder
	for _, src := range corpus(t) {
		if !strings.HasSuffix(src.Rel, "bug-ledger.md") {
			all.WriteString(src.Body)
		}
	}
	names := helpEnvLine.FindAllStringSubmatch(section, -1)
	if len(names) < 20 {
		t.Fatalf("found %d variables in ompig --help; the parser is broken", len(names))
	}
	for _, m := range names {
		if !strings.Contains(all.String(), m[1]) {
			t.Errorf("%s is listed by `ompig --help` and documented nowhere", m[1])
		}
	}
}

var docsShowRef = regexp.MustCompile("ompig docs show ([a-z0-9][a-z0-9.-]*)")

// A `ompig docs show <topic>` the docs print must open a page of the embedded bundle.
func TestDocumentedDocsTopicsExist(t *testing.T) {
	out, code := runPig(t, "docs", "list")
	if code != 0 {
		t.Fatalf("ompig docs list exits %d:\n%s", code, out)
	}
	topics := map[string]bool{}
	for f := range strings.FieldsSeq(out) {
		topics[strings.TrimSuffix(f, ".md")] = true
	}
	checked := 0
	for _, src := range corpus(t) {
		for i, line := range strings.Split(src.Body, "\n") {
			for _, m := range docsShowRef.FindAllStringSubmatch(line, -1) {
				checked++
				if !topics[strings.TrimSuffix(m[1], ".md")] {
					t.Errorf("%s:%d: `ompig docs show %s` names no page of the bundle; `ompig docs list` has %d pages", src.Rel, i+1, m[1], len(topics))
				}
			}
		}
	}
	if checked < 5 {
		t.Fatalf("found only %d `ompig docs show` references; the pattern is broken", checked)
	}
}

// Each command the top-level help lists is in the CLI reference.
func TestEveryHelpCommandIsInTheCLIReference(t *testing.T) {
	help := helpOf(t)
	commands := regexp.MustCompile(`(?m)^  ompig ([a-z][a-z-]*)\b`).FindAllStringSubmatch(help, -1)
	if len(commands) < 10 {
		t.Fatalf("found %d commands in ompig --help; the parser is broken", len(commands))
	}
	for _, src := range corpus(t) {
		if src.Rel != "docs/site/docs/cli.md" && src.Rel != "internal/pigdocs/content/commands.md" {
			continue
		}
		for _, m := range commands {
			if !strings.Contains(src.Body, "ompig "+m[1]) {
				t.Errorf("%s does not document `ompig %s`, which `ompig --help` lists", src.Rel, m[1])
			}
		}
	}
}

// firstCells returns the first cell of every table row that starts with a `ompig` command or an option.
func firstCells(body string) map[string]bool {
	out := map[string]bool{}
	cell := regexp.MustCompile(`^\| ((?:` + "`[^`]*`" + `|[^|` + "`" + `]|\\\|)+?) \| `)
	for line := range strings.SplitSeq(body, "\n") {
		m := cell.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		c := strings.TrimSpace(m[1])
		if strings.HasPrefix(c, "`ompig ") || strings.HasPrefix(c, "`--") || strings.HasPrefix(c, "`-") {
			out[c] = true
		}
	}
	return out
}

// The site's CLI reference and the bundle's command page describe the same commands and options, and each was last edited by someone who did not touch the other. Their tables must list the same commands and options.
func TestCLIReferencePagesListTheSameCommandsAndOptions(t *testing.T) {
	var site, bundle string
	for _, src := range corpus(t) {
		switch src.Rel {
		case "docs/site/docs/cli.md":
			site = src.Body
		case "internal/pigdocs/content/commands.md":
			bundle = src.Body
		}
	}
	a, b := firstCells(site), firstCells(bundle)
	if len(a) < 40 || len(b) < 40 {
		t.Fatalf("read %d site rows and %d bundle rows; the table reader is broken", len(a), len(b))
	}
	for c := range a {
		if !b[c] {
			t.Errorf("internal/pigdocs/content/commands.md has no row for %s, which docs/site/docs/cli.md lists", c)
		}
	}
	for c := range b {
		if !a[c] {
			t.Errorf("docs/site/docs/cli.md has no row for %s, which internal/pigdocs/content/commands.md lists", c)
		}
	}
}

// The verbs `ompig ompiglet` prints are the verbs the reference names in its Ompiglet row.
func TestPigletVerbRowListsEveryVerb(t *testing.T) {
	help := helpOf(t, "ompiglet")
	verbs := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^  ompig ompiglet ([a-z]+)`).FindAllStringSubmatch(help, -1) {
		verbs[m[1]] = true
	}
	if len(verbs) < 10 {
		t.Fatalf("found %d verbs in ompig ompiglet help; the parser is broken", len(verbs))
	}
	for _, src := range corpus(t) {
		if src.Rel != "docs/site/docs/cli.md" && src.Rel != "internal/pigdocs/content/commands.md" {
			continue
		}
		var row string
		for line := range strings.SplitSeq(src.Body, "\n") {
			if strings.HasPrefix(line, "| `ompig ompiglet list") {
				row = line
			}
		}
		if row == "" {
			t.Fatalf("%s has no row listing the ompiglet verbs", src.Rel)
		}
		for v := range verbs {
			if !strings.Contains(row, v) {
				t.Errorf("%s: the ompiglet verbs row omits %q, which `ompig ompiglet` lists", src.Rel, v)
			}
		}
	}
}
