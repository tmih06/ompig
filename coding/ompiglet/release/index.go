// Package release verifies and installs signed Ompiglet Binary releases.
//
// ompig additive (D18): Ompiglet releases are product-neutral signed indexes whose
// per-target binaries can be pulled without contacting a Pi or Ompig service.
package release

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/tmih06/ompig/coding/ompiglet/signature"
)

// PayloadType is the DSSE payload type of a Ompiglet release index.
const PayloadType = "application/vnd.ompig.ompiglet-release+json"

// Index names one Ompiglet release and its available target binaries.
type Index struct {
	Ompiglet     string            `json:"ompiglet"`
	Version      string            `json:"version"`
	OmpigVersion string            `json:"ompigVersion"`
	SourceRef    string            `json:"sourceRef"`
	GitHub       *GitHubRelease    `json:"github,omitempty"`
	Signer       signature.Signer  `json:"signer"`
	Binaries     map[string]Binary `json:"binaries"`
}

// Binary is one downloadable signed Ompiglet Binary. SHA256 covers the complete
// downloaded file, including its signature trailer.
type Binary struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// VerifiedIndex is an index whose embedded public key verified its DSSE
// envelope. The key is still subject to the caller's trust and continuity
// policy.
type VerifiedIndex struct {
	Index     Index
	Envelope  signature.Envelope
	PublicKey ed25519.PublicKey
}

// Sign signs index with key and returns its DSSE envelope as JSON.
func Sign(index Index, key ed25519.PrivateKey) ([]byte, error) {
	public, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Ompiglet release signing key is not ed25519")
	}
	index.Signer = signature.Signer{
		KeyID:     signature.KeyID(public),
		PublicKey: base64.StdEncoding.EncodeToString(public),
	}
	if err := Validate(index); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}
	envelope := signature.SignEnvelope(PayloadType, payload, key)
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Verify strictly decodes a release envelope, validates its index, and checks
// the DSSE signature with the public key named by the signed index.
func Verify(data []byte) (VerifiedIndex, error) {
	envelope, err := signature.ParseEnvelope(data)
	if err != nil {
		return VerifiedIndex{}, fmt.Errorf("Ompiglet release index: %w", err)
	}
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return VerifiedIndex{}, fmt.Errorf("Ompiglet release index payload: %w", err)
	}
	var index Index
	if err := strictJSON(payload, &index); err != nil {
		return VerifiedIndex{}, fmt.Errorf("Ompiglet release index payload: %w", err)
	}
	if err := Validate(index); err != nil {
		return VerifiedIndex{}, err
	}
	public, err := base64.StdEncoding.DecodeString(index.Signer.PublicKey)
	if err != nil || len(public) != ed25519.PublicKeySize || index.Signer.KeyID != signature.KeyID(public) {
		return VerifiedIndex{}, fmt.Errorf("Ompiglet release index names an invalid ed25519 public key")
	}
	if _, err := envelope.Verify(PayloadType, ed25519.PublicKey(public)); err != nil {
		return VerifiedIndex{}, fmt.Errorf("Ompiglet release index: %w", err)
	}
	return VerifiedIndex{Index: index, Envelope: envelope, PublicKey: ed25519.PublicKey(public)}, nil
}

// Validate checks the strict current shape of a Ompiglet release index.
func Validate(index Index) error {
	if !validPathPart(index.Ompiglet) {
		return fmt.Errorf("Ompiglet release index has invalid Ompiglet name %q", index.Ompiglet)
	}
	if !validReleaseVersion(index.Version) {
		return fmt.Errorf("Ompiglet release index has invalid version %q", index.Version)
	}
	if index.OmpigVersion == "" || index.OmpigVersion != strings.TrimSpace(index.OmpigVersion) {
		return fmt.Errorf("Ompiglet release index has invalid Ompig version %q", index.OmpigVersion)
	}
	if index.SourceRef == "" || index.SourceRef != strings.TrimSpace(index.SourceRef) {
		return fmt.Errorf("Ompiglet release index has invalid sourceRef %q", index.SourceRef)
	}
	if index.GitHub != nil {
		if err := index.GitHub.validate(index.Ompiglet); err != nil {
			return err
		}
	}
	if index.Signer.KeyID == "" || index.Signer.PublicKey == "" {
		return fmt.Errorf("Ompiglet release index signer is required")
	}
	if len(index.Binaries) == 0 {
		return fmt.Errorf("Ompiglet release index has no binaries")
	}
	for target, binary := range index.Binaries {
		if !validTarget(target) {
			return fmt.Errorf("Ompiglet release index has invalid target %q", target)
		}
		parsed, err := url.Parse(binary.URL)
		if err != nil || binary.URL == "" || parsed.Fragment != "" {
			return fmt.Errorf("Ompiglet release target %s has invalid URL", target)
		}
		if len(binary.SHA256) != 64 {
			return fmt.Errorf("Ompiglet release target %s has invalid SHA256 %q", target, binary.SHA256)
		}
		decoded, err := hex.DecodeString(binary.SHA256)
		if err != nil || hex.EncodeToString(decoded) != binary.SHA256 {
			return fmt.Errorf("Ompiglet release target %s has invalid SHA256 %q", target, binary.SHA256)
		}
		if binary.Size <= 0 {
			return fmt.Errorf("Ompiglet release target %s has invalid size %d", target, binary.Size)
		}
	}
	return nil
}

func validReleaseVersion(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && semver.IsValid("v"+value)
}

func validTarget(target string) bool {
	goos, goarch, ok := strings.Cut(target, "/")
	return ok && validPathPart(goos) && validPathPart(goarch) && !strings.Contains(goarch, "/")
}

func validPathPart(value string) bool {
	return value != "" && value != "." && value != ".." && value == strings.TrimSpace(value) && !strings.ContainsAny(value, `/\\`)
}

func strictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing content")
	}
	return nil
}
