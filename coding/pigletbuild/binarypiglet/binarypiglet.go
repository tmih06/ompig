// Package binarypiglet holds the Ompiglet and resolution record embedded in a
// Ompiglet Binary. A Ompiglet Binary
// always bakes the ompiglet fields needed to assemble the agent (extensions,
// skills, MCP, scoping, discovery) and, for a built Ompiglet Binary, its
// resolution record so the binary can verify its own build identity at startup.
// A signed build also embeds its author's signer public keys in signers.txt
// and carries a signature block after the executable bytes (see
// coding/ompiglet/signature). Stock Ompig ships ompiglet.yaml,
// resolution-record.json, and signers.txt empty: Default returns nil, Verify
// is a no-op, and startup resolves the ompiglet exactly as before.
package binarypiglet

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/tmih06/ompig/coding"
	ompiglet "github.com/tmih06/ompig/coding/ompiglet"
	"github.com/tmih06/ompig/coding/ompiglet/artifact"
	"github.com/tmih06/ompig/coding/ompiglet/signature"
	"github.com/tmih06/ompig/internal/codingagent"
)

//go:embed ompiglet.yaml
var pigletYAML []byte

//go:embed resolution-record.json
var resolutionRecordJSON []byte

//go:embed signers.txt
var signersPEM []byte

// executable locates the running binary whose signature block is checked.
var executable = os.Executable

// Default returns the baked ompiglet, or nil when none was baked. It is
// intentionally conservative: any parse failure, or a baked ompiglet with only a
// name and no agent-defining fields, yields nil so the caller's normal ompiglet
// resolution runs unchanged.
func Default() *ompiglet.Ompiglet { return parse(pigletYAML) }

// parse decodes baked ompiglet YAML, returning nil on any parse failure or when
// the ompiglet carries no agent-defining fields.
func parse(data []byte) *ompiglet.Ompiglet {
	p, err := ompiglet.ParseBytes(data)
	if err != nil || p == nil {
		return nil
	}
	// ompig additive (D92): a strip list alone defines a Ompiglet Binary's agent.
	if p.Model == nil && p.SystemPrompt == nil && len(p.Extensions) == 0 && len(p.Skills) == 0 && p.BuiltinTools == nil && p.Discovery == nil && p.Strip.IsEmpty() {
		return nil
	}
	return p
}

// OverlayBase returns the baked Ompiglet as the base of an overlay when this Ompiglet Binary accepts one (its Ompiglet set
// build.allowOverlays), and nil otherwise, including for stock ompig. The flag is part of the baked Ompiglet, so the digest the
// record and the signed manifest name covers it.
//
// ompig additive (D202): a Ompiglet Binary can declare that it accepts overlay Ompiglets.
func OverlayBase() *ompiglet.OverlayBase {
	if !IsPigletBinary() {
		return nil
	}
	p, err := ompiglet.ParseBytes(pigletYAML)
	if err != nil || p.Build == nil || !p.Build.AllowOverlays {
		return nil
	}
	return &ompiglet.OverlayBase{Ompiglet: p, Digest: "sha256:" + hex.EncodeToString(sha256Sum(pigletYAML))}
}

// Name returns the name of the Ompiglet baked into this Binary, or "" for stock ompig.
func Name() string {
	record, err := DefaultClosure()
	if err != nil || record == nil {
		return ""
	}
	return record.Ompiglet
}

// DefaultClosure returns the Ompiglet resolution record baked into this binary,
// or nil when none was baked (stock ompig). A non-empty but malformed closure is
// a fail-closed condition: it returns an error rather than a nil record so the
// caller refuses to run a tampered binary.
func DefaultClosure() (*artifact.Record, error) {
	trimmed := bytes.TrimSpace(resolutionRecordJSON)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var record artifact.Record
	if err := json.Unmarshal(trimmed, &record); err != nil {
		return nil, fmt.Errorf("decode baked Ompiglet closure: %w", err)
	}
	return &record, nil
}

// IsPigletBinary reports whether this executable was built as a Ompiglet Binary
// bound to one baked Ompiglet. It keys on the presence of a baked closure, so a
// stock ompig build (no closure) is not a Ompiglet Binary.
func IsPigletBinary() bool {
	return len(bytes.TrimSpace(resolutionRecordJSON)) > 0
}

// Verify checks that the closure baked into a Ompiglet Binary is a valid,
// untampered resolution record whose effective digest matches the baked
// Ompiglet, and checks its signature. Stock ompig (no baked closure) verifies
// trivially. A built Ompiglet Binary that fails these checks must not run: its
// recorded build identity or its embedded Ompiglet has been altered. A signed
// Binary whose unchanged file passed before skips re-hashing its executable
// bytes (see signature.CheckCached). cacheErr reports that a passing check
// could not be remembered; the Binary still runs, and its next start checks
// it in full again.
func Verify() (cacheErr, err error) {
	record, err := DefaultClosure()
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}
	if record.Kind != artifact.RecordKindResolution || record.Resolution == nil {
		return nil, fmt.Errorf("baked Ompiglet closure is not a resolution record")
	}
	if err := artifact.ValidateRecord(*record); err != nil {
		return nil, fmt.Errorf("baked Ompiglet closure failed verification: %w", err)
	}
	pigletDigest := "sha256:" + hex.EncodeToString(sha256Sum(pigletYAML))
	if pigletDigest != record.Resolution.EffectiveDigest {
		return nil, fmt.Errorf("baked Ompiglet does not match its closure: Ompiglet digest %s, closure expects %s", pigletDigest, record.Resolution.EffectiveDigest)
	}
	return checkSignature(record)
}

// checkSignature checks this Ompiglet Binary's signature block against the
// author's embedded signer keys and the user's Ompiglet trust store, offline.
// An unsigned binary passes unless its author embedded a signer or the user's
// policy requires a signature. A signed manifest must also name this binary's
// own record, Ompiglet, target, and Ompig version.
// ompig additive (D18): signed Ompiglet Binaries refuse to start when altered.
func checkSignature(record *artifact.Record) (cacheErr, err error) {
	embedded, err := signature.ParsePublicKeys(signersPEM)
	if err != nil {
		return nil, fmt.Errorf("baked Ompiglet signer keys: %w", err)
	}
	trust, err := signature.LoadTrust(signature.TrustDir())
	if err != nil {
		return nil, fmt.Errorf("Ompiglet trust store: %w", err)
	}
	self, err := executable()
	if err != nil {
		return nil, fmt.Errorf("locate this Ompiglet Binary to check its signature: %w", err)
	}
	// ompig additive (D18): the verification cache spares an unchanged Binary the hash of its executable bytes; it is written only after every check passed.
	check, err := signature.CheckCached(self, signature.Policy{Trust: trust, Embedded: embedded, RequireKnownSigner: true}, verifyCachePath())
	if err != nil || !check.Status.Signed {
		return nil, err
	}
	if err := matchManifest(check.Status.Manifest, record); err != nil {
		return nil, err
	}
	return check.Remember(), nil
}

// verifyCachePath is the per-user file that remembers passing startup checks.
func verifyCachePath() string {
	return filepath.Join(codingagent.StateDir("ompiglet-verify"), "verified.json")
}

// matchManifest binds the verified signed manifest to what this binary
// actually runs, so a validly signed manifest cannot vouch for another build.
func matchManifest(manifest signature.Manifest, record *artifact.Record) error {
	pigletDigest := "sha256:" + hex.EncodeToString(sha256Sum(pigletYAML))
	for _, check := range []struct{ name, signed, actual string }{
		{"resolution record", manifest.ResolutionDigest, record.Digest},
		{"Ompiglet definition", manifest.PigletDigest, pigletDigest},
		{"component plan", manifest.ComponentPlanDigest, record.Resolution.ComponentPlan.Digest},
		{"target", manifest.Target, runtime.GOOS + "/" + runtime.GOARCH},
		{"Ompig version", manifest.OmpigVersion, coding.OmpigVersion},
	} {
		if check.signed != check.actual {
			return fmt.Errorf("signed Ompiglet manifest names %s %q, but this binary has %q", check.name, check.signed, check.actual)
		}
	}
	return nil
}

func sha256Sum(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

// VerifyRegisteredClosure checks that the fused and binary-packed extension
// components a built Ompiglet Binary actually registers at startup match its
// baked plan. A tampered fuse registry or cell manifest, or a PATH/ambient
// substitute for a component the plan says was built in, fails closed. Stock
// ompig (no baked closure) is a no-op. Components the plan supplies from a
// release, agentEnv, or an external service are not carried in the binary and
// are excluded from this check.
func VerifyRegisteredClosure(fusedNames, packedNames []string) error {
	record, err := DefaultClosure()
	if err != nil {
		return err
	}
	if record == nil {
		return nil
	}
	if record.Resolution == nil {
		return fmt.Errorf("baked Ompiglet closure has no resolution payload")
	}
	var wantFused, wantPacked []string
	for _, component := range record.Resolution.ComponentPlan.Components {
		if component.Kind != artifact.ComponentKindExtension {
			continue
		}
		switch {
		case component.Realization == artifact.RealizationFused:
			wantFused = append(wantFused, component.Name)
		case component.Realization == artifact.RealizationSubprocess && component.Materialization == artifact.MaterializationBinary:
			wantPacked = append(wantPacked, component.Name)
		}
	}
	if err := equalNameSets("fused", wantFused, fusedNames); err != nil {
		return err
	}
	return equalNameSets("binary-packed", wantPacked, packedNames)
}

// equalNameSets fails closed when the built-in components and the planned
// components differ in either direction.
func equalNameSets(label string, want, got []string) error {
	wantCount := map[string]int{}
	for _, name := range want {
		wantCount[name]++
	}
	gotCount := map[string]int{}
	for _, name := range got {
		gotCount[name]++
	}
	var missing, unexpected []string
	for name, count := range wantCount {
		if gotCount[name] < count {
			missing = append(missing, name)
		}
	}
	for name, count := range gotCount {
		if wantCount[name] < count {
			unexpected = append(unexpected, name)
		}
	}
	if len(missing) == 0 && len(unexpected) == 0 {
		return nil
	}
	slices.Sort(missing)
	slices.Sort(unexpected)
	return fmt.Errorf("baked Ompiglet Binary %s extensions do not match its plan: missing %v, unexpected %v", label, missing, unexpected)
}
