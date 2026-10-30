package ompiglet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	sourceref "github.com/tmih06/ompig/coding/source"
)

// NPMKeyword is the npm keyword that lists a package as a Ompiglet source. Catalogs find published Ompiglets by this keyword the way Pi's gallery finds Pi packages by `pi-package`.
const NPMKeyword = "ompig-ompiglet"

// authorMetadata lists the package.json fields a Ompiglet's own package.json can supply to its published package. Every other field is dropped so scripts, dependencies, and lifecycle hooks of an authoring repository never reach the generated source.
var authorMetadata = []string{"license", "author", "repository", "homepage", "bugs", "funding", "publishConfig"}

// NPMBinaries names the signed Ompiglet Binary release that matches a published npm source. Catalogs show `ompig ompiglet pull <Ref>` next to the source install command.
type NPMBinaries struct {
	Ref    string `json:"ref"`
	Signer string `json:"signer"`
}

// NPMSourceOptions selects the authored Ompiglet and the publication choices that its YAML cannot express.
type NPMSourceOptions struct {
	// Path is the authored Ompiglet YAML file.
	Path string
	// Workspace anchors workspace-bound Ompiglet inputs during validation.
	Workspace string
	// NPMName overrides the published package name. The default is the name in a package.json beside the Ompiglet, then the Ompiglet's own name.
	NPMName string
	// PackageMap maps a Package alias to the npm reference that replaces its source.
	PackageMap map[string]string
	// Binaries is recorded in the generated package.json when the release has signed Binaries.
	Binaries *NPMBinaries
}

// NPMSource is one generated, publishable npm package directory.
type NPMSource struct {
	// Dir is a new temporary directory. The caller removes it.
	Dir     string
	Name    string
	Version string
	// Packages maps each Package alias to its source in the published Ompiglet.
	Packages map[string]string
	// Files lists every file in Dir, slash-separated and sorted.
	Files    []string
	Warnings []string
}

// npmPigField is the `ompig` block of a published Ompiglet's package.json. `ompiglet` is the path that `ompig ompiglet add` reads.
type npmPigField struct {
	Ompiglet string       `json:"ompiglet"`
	Binaries *NPMBinaries `json:"binaries,omitempty"`
}

// npmManifest is the generated package.json. Field order is the order npm and catalogs read comfortably.
type npmManifest struct {
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	Description   string          `json:"description,omitempty"`
	Keywords      []string        `json:"keywords"`
	License       json.RawMessage `json:"license,omitempty"`
	Author        json.RawMessage `json:"author,omitempty"`
	Repository    json.RawMessage `json:"repository,omitempty"`
	Homepage      json.RawMessage `json:"homepage,omitempty"`
	Bugs          json.RawMessage `json:"bugs,omitempty"`
	Funding       json.RawMessage `json:"funding,omitempty"`
	PublishConfig json.RawMessage `json:"publishConfig,omitempty"`
	Files         []string        `json:"files"`
	Ompig         npmPigField     `json:"ompig"`
}

// CheckRemoteAddable applies the checks `ompig ompiglet add npm:<package>` makes to published Ompiglet source that need neither network nor materialized Packages. It guarantees that a source which passes can be registered from npm.
//
// ompig additive (D18): npm Ompiglet source publication.
func CheckRemoteAddable(path string) error {
	p, err := Parse(path)
	if err != nil {
		return err
	}
	return checkRemoteAddable(p)
}

func checkRemoteAddable(p *Ompiglet) error {
	if p.Extends != nil {
		return fmt.Errorf("Ompiglet add cannot copy an extends dependency closure; run the Ompiglet from its source path")
	}
	if p.AgentEnv != nil && p.AgentEnv.DevContainer != "" {
		return fmt.Errorf("Ompiglet add cannot copy an agentEnv.devContainer closure; run the Ompiglet from its source path")
	}
	if err := validatePortableCatalogPiglet(p, false); err != nil {
		return err
	}
	if err := validatePigletAddOrigins(p); err != nil {
		return err
	}
	return validatePigletAddPackageSources(p)
}

// BuildNPMSource validates an authored Ompiglet and writes the npm package that publishes it into a new temporary directory.
//
// The generated ompiglet.yaml is the authored file with every local Package replaced by the npm reference of that Package, because a remote Ompiglet source cannot take local paths. The generated package.json carries the Ompiglet's name and release.version, the `ompig-ompiglet` keyword, and the `ompig.ompiglet` path that `ompig ompiglet add` reads. The authored files are not modified.
//
// ompig additive (D18): npm Ompiglet source publication.
func BuildNPMSource(options NPMSourceOptions) (source NPMSource, err error) {
	authored, err := os.ReadFile(options.Path)
	if err != nil {
		return NPMSource{}, fmt.Errorf("read Ompiglet %s: %w", options.Path, err)
	}
	p, err := Parse(options.Path)
	if err != nil {
		return NPMSource{}, err
	}
	if err := checkPublishable(p); err != nil {
		return NPMSource{}, err
	}
	// Validate the authored Ompiglet exactly as `ompig ompiglet validate` does: every origin resolves.
	if _, err := ResolveEffectiveWithOptions(options.Path, ResolveOptions{Workspace: options.Workspace}); err != nil {
		return NPMSource{}, fmt.Errorf("Ompiglet %q does not validate: %w", p.Name, err)
	}
	rewritten, err := publishedPackages(p, options.PackageMap)
	if err != nil {
		return NPMSource{}, err
	}
	document, err := rewritePackages(authored, rewritten)
	if err != nil {
		return NPMSource{}, err
	}
	published, err := ParseBytes(document)
	if err != nil {
		return NPMSource{}, fmt.Errorf("generated Ompiglet source is invalid: %w", err)
	}
	if err := checkRemoteAddable(published); err != nil {
		return NPMSource{}, fmt.Errorf("Ompiglet %q cannot be published as npm source: %w", p.Name, err)
	}
	dir := filepath.Dir(p.SourcePath())
	author, err := readAuthorPackage(dir)
	if err != nil {
		return NPMSource{}, err
	}
	name, err := npmName(options.NPMName, author.name, p.Name)
	if err != nil {
		return NPMSource{}, err
	}
	stage, err := os.MkdirTemp("", "ompig-ompiglet-npm-*")
	if err != nil {
		return NPMSource{}, fmt.Errorf("create npm source directory: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(stage)
		}
	}()
	result := NPMSource{Dir: stage, Name: name, Version: p.Release.Version, Packages: rewritten}
	files := map[string][]byte{
		"ompiglet.yaml": append([]byte("# Generated by ompig ompiglet publish from "+filepath.Base(options.Path)+". Edit the authored Ompiglet, not this file.\n"), document...),
	}
	if p.SystemPrompt != nil && p.SystemPrompt.File != "" {
		relative, data, err := readPigletRelativeFile(p.SourcePath(), p.SystemPrompt.File)
		if err != nil {
			return NPMSource{}, fmt.Errorf("systemPrompt.file %q: %w", p.SystemPrompt.File, err)
		}
		files[filepath.ToSlash(relative)] = data
	}
	for _, candidate := range []string{"README.md", "README"} {
		if _, data, err := readPigletRelativeFile(p.SourcePath(), candidate); err == nil {
			files[candidate] = data
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return NPMSource{}, fmt.Errorf("%s: %w", candidate, err)
		}
	}
	if !hasFile(files, "README.md", "README") {
		files["README.md"] = []byte(generatedREADME(name, p.Description))
	}
	for _, candidate := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt"} {
		if _, data, err := readPigletRelativeFile(p.SourcePath(), candidate); err == nil {
			files[candidate] = data
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return NPMSource{}, fmt.Errorf("%s: %w", candidate, err)
		}
	}
	if !hasFile(files, "LICENSE", "LICENSE.md", "LICENSE.txt") {
		result.Warnings = append(result.Warnings, "no LICENSE file beside the Ompiglet; add one so users know the terms")
	}
	manifest := npmManifest{
		Name: name, Version: p.Release.Version, Description: p.Description,
		Keywords: mergeKeywords(author.keywords),
		Ompig:    npmPigField{Ompiglet: "ompiglet.yaml", Binaries: options.Binaries},
	}
	for _, field := range authorMetadata {
		setMetadata(&manifest, field, author.fields[field])
	}
	manifest.Files = slices.Sorted(maps.Keys(files))
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return NPMSource{}, err
	}
	files["package.json"] = append(data, '\n')
	for relative, content := range files {
		target := filepath.Join(stage, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return NPMSource{}, err
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return NPMSource{}, err
		}
	}
	result.Files = slices.Sorted(maps.Keys(files))
	return result, nil
}

func hasFile(files map[string][]byte, names ...string) bool {
	return slices.ContainsFunc(names, func(name string) bool { _, ok := files[name]; return ok })
}

// checkPublishable refuses what no npm source can carry before any resolution runs.
func checkPublishable(p *Ompiglet) error {
	if p.Extends != nil {
		return fmt.Errorf("Ompiglet %q uses extends, which a published npm source cannot carry (ompig ompiglet add refuses it); flatten the derived Ompiglet into one file before publishing", p.Name)
	}
	if p.Release == nil || p.Release.Version == "" {
		return fmt.Errorf("Ompiglet %q needs release.version to be published", p.Name)
	}
	if !sourceref.ValidNPMVersion(p.Release.Version) {
		return fmt.Errorf("Ompiglet %q release.version %q is not a version npm publishes; use a full semantic version such as 1.2.3", p.Name, p.Release.Version)
	}
	local := func(kind, name string, origins []string) error {
		for _, origin := range origins {
			if strings.HasPrefix(origin, "package:") {
				continue
			}
			if ref, err := validateTypedSource(origin, sourceref.BareReject); err == nil && ref.Kind == sourceref.KindLocal {
				return fmt.Errorf("%s %q has the local origin %q; a published Ompiglet takes Resources only from its Packages (package:<alias>), npm, or Git. Move it into a Package and publish that Package first", kind, name, origin)
			}
		}
		return nil
	}
	for _, extension := range p.Extensions {
		if err := local("extension", extension.Name, extension.Origins); err != nil {
			return err
		}
	}
	for _, skill := range p.Skills {
		if err := local("skill", skill.Name, skill.Origins); err != nil {
			return err
		}
	}
	return nil
}

// publishedPackages returns the source each Package alias has in the published Ompiglet.
func publishedPackages(p *Ompiglet, packageMap map[string]string) (map[string]string, error) {
	for _, alias := range slices.Sorted(maps.Keys(packageMap)) {
		if _, declared := p.Packages[alias]; !declared {
			return nil, fmt.Errorf("--package-map names %q, which the Ompiglet does not declare in packages", alias)
		}
		ref, err := validateTypedSource(packageMap[alias], sourceref.BareReject)
		if err != nil || ref.Kind != sourceref.KindNPM {
			return nil, fmt.Errorf("--package-map %s=...: the value must be an npm: reference such as npm:<name>@<range>", alias)
		}
		if ref.NPMVer == "" {
			return nil, fmt.Errorf("--package-map %s=%s needs a version range such as npm:%s@^1.0.0", alias, packageMap[alias], ref.NPMName)
		}
	}
	out := make(map[string]string, len(p.Packages))
	for _, alias := range slices.Sorted(maps.Keys(p.Packages)) {
		source := p.Packages[alias]
		if mapped, ok := packageMap[alias]; ok {
			out[alias] = mapped
			continue
		}
		ref, err := validateTypedSource(source, sourceref.BareReject)
		if err != nil || ref.Kind != sourceref.KindLocal {
			out[alias] = source
			continue
		}
		root := expandOriginPath(p.sourceDir, ref.Locator)
		identity, err := localPackageIdentity(root)
		if err != nil {
			return nil, fmt.Errorf("Package %q is the local source %q, which cannot be named on npm (%w); publish it first (ompig package publish %s --to npm --yes) so the Ompiglet can use npm:<name>@^<version>, or pass its npm reference with --package-map %s=npm:<name>@<range>", alias, source, err, ref.Locator, alias)
		}
		out[alias] = "npm:" + identity
	}
	return out, nil
}

// localPackageIdentity returns "<name>@^<version>" for the Package at root, followed by "?registry=<url>" when its package.json publishes to publishConfig.registry, so the published Ompiglet installs the Package from the registry that has it.
func localPackageIdentity(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return "", errors.New("it has no package.json")
	}
	var manifest struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		Private       bool   `json:"private"`
		PublishConfig struct {
			Registry string `json:"registry"`
		} `json:"publishConfig"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("its package.json is not valid JSON: %w", err)
	}
	switch {
	case manifest.Private:
		return "", errors.New("its package.json is private")
	case manifest.Name == "":
		return "", errors.New("its package.json has no name")
	case !sourceref.ValidNPMPackageName(manifest.Name):
		return "", fmt.Errorf("its package.json name %q is not an npm package name", manifest.Name)
	case manifest.Version == "":
		return "", errors.New("its package.json has no version")
	case !sourceref.ValidNPMVersion(manifest.Version):
		return "", fmt.Errorf("its package.json version %q is not a version npm publishes", manifest.Version)
	}
	identity := manifest.Name + "@^" + manifest.Version
	if manifest.PublishConfig.Registry == "" {
		return identity, nil
	}
	ref, err := sourceref.Parse("npm:"+identity+"?registry="+url.QueryEscape(manifest.PublishConfig.Registry), sourceref.Options{Bare: sourceref.BareReject})
	if err != nil {
		return "", fmt.Errorf("its package.json publishConfig.registry %q cannot be named in an npm: source (%w)", manifest.PublishConfig.Registry, err)
	}
	return identity + "?registry=" + url.QueryEscape(ref.NPMRegistry), nil
}

// rewritePackages replaces the value of each declared Package in the YAML document and keeps the authored comments and layout.
func rewritePackages(authored []byte, packages map[string]string) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(authored, &document); err != nil {
		return nil, fmt.Errorf("parse Ompiglet YAML: %w", err)
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("Ompiglet YAML is not a mapping")
	}
	root := document.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "packages" {
			continue
		}
		table := root.Content[i+1]
		for j := 0; j+1 < len(table.Content); j += 2 {
			alias := table.Content[j].Value
			if replacement, ok := packages[alias]; ok && table.Content[j+1].Kind == yaml.ScalarNode {
				table.Content[j+1].Value = replacement
			}
		}
	}
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, fmt.Errorf("write Ompiglet YAML: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type authorPackage struct {
	name     string
	keywords []string
	fields   map[string]json.RawMessage
}

// readAuthorPackage reads the optional package.json beside the Ompiglet. It supplies metadata only.
func readAuthorPackage(dir string) (authorPackage, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return authorPackage{}, nil
	}
	if err != nil {
		return authorPackage{}, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return authorPackage{}, fmt.Errorf("package.json beside the Ompiglet is not valid JSON: %w", err)
	}
	var out authorPackage
	out.fields = fields
	_ = json.Unmarshal(fields["name"], &out.name)
	_ = json.Unmarshal(fields["keywords"], &out.keywords)
	return out, nil
}

func npmName(override, authored, ompiglet string) (string, error) {
	name := firstNonEmpty(override, authored, ompiglet)
	if !sourceref.ValidNPMPackageName(name) {
		return "", fmt.Errorf("%q is not an npm package name (lowercase, optionally scoped as @scope/name); choose one with --npm-name", name)
	}
	return name, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func mergeKeywords(authored []string) []string {
	out := []string{NPMKeyword}
	for _, keyword := range authored {
		if keyword != "" && !slices.Contains(out, keyword) {
			out = append(out, keyword)
		}
	}
	return out
}

func setMetadata(manifest *npmManifest, field string, value json.RawMessage) {
	if len(value) == 0 || string(value) == "null" {
		return
	}
	switch field {
	case "license":
		manifest.License = value
	case "author":
		manifest.Author = value
	case "repository":
		manifest.Repository = value
	case "homepage":
		manifest.Homepage = value
	case "bugs":
		manifest.Bugs = value
	case "funding":
		manifest.Funding = value
	case "publishConfig":
		manifest.PublishConfig = value
	}
}

func generatedREADME(name, description string) string {
	var out strings.Builder
	out.WriteString("# " + name + "\n\n")
	if description != "" {
		out.WriteString(description + "\n\n")
	}
	out.WriteString("A Ompiglet published for Ompig.\n\n## Install\n\n```sh\nompig ompiglet add npm:" + name + "\n```\n")
	return out.String()
}
