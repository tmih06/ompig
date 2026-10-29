package ompiglet

import (
	"errors"
	"strings"

	sourceref "github.com/tmih06/ompig/coding/source"
)

// DistributionOffline reports whether PI_OFFLINE or OMPIG_OFFLINE forbids the
// network access that remote Ompiglet distribution commands need.
func DistributionOffline() bool {
	return remotePigletAddOffline()
}

// ValidateRemoteSource checks that raw is an npm or Git source reference that carries no authentication material, so it can be published. Rejected references and parser diagnostics never enter its errors.
func ValidateRemoteSource(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("source is required")
	}
	ref, err := sourceref.Parse(raw, sourceref.Options{Bare: sourceref.BareReject})
	if err != nil {
		// ompig additive (D18): shared parser errors may contain credentials from a rejected reference; distribution returns only a static diagnostic.
		return errors.New("invalid remote Ompiglet source: use an explicit scheme (npm: or git:); malformed references and unsupported source schemes are refused")
	}
	return validateRemotePigletOriginSource(ref)
}
