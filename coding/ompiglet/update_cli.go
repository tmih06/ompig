package ompiglet

import (
	"context"
	"fmt"
	"io"
	"slices"

	pigletrelease "github.com/tmih06/ompig/coding/ompiglet/release"
)

const updateUsage = "Usage: ompig ompiglet update <name> [--version version] [--target os/arch] [--accept-signer key-id] [--json] [--no-input]"

// ompig additive (D18): named Binary updates use the installed signed repository and tag namespace, never the repository's latest release.
func cmdUpdate(args []string, stdout, stderr io.Writer) int {
	jsonMode := slices.Contains(args, "--json")
	name, target, version, accepted, err := parseReleaseArgs(args, "update")
	if err != nil {
		if jsonMode {
			writePigletCommandJSON(pigletCommandOutput{Command: "update", Error: err.Error()}, stdout)
		} else {
			_, _ = fmt.Fprintf(stderr, "error: %v\n%s\n", err, updateUsage)
		}
		return 2
	}
	if name == "" {
		_, _ = fmt.Fprintln(stdout, updateUsage)
		return 0
	}
	if DistributionOffline() {
		err = fmt.Errorf("cannot update Ompiglet while offline; unset OMPIG_OFFLINE and PI_OFFLINE")
	}
	var result pigletrelease.Result
	if err == nil {
		result, err = pigletrelease.Update(context.Background(), name, pigletrelease.Options{Target: target, Version: version, AcceptSigner: accepted})
	}
	if err != nil {
		if jsonMode {
			writePigletCommandJSON(pigletCommandOutput{Command: "update", Error: err.Error()}, stdout)
		} else {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		}
		return 1
	}
	message := fmt.Sprintf("Current %s %s for %s, signed by %s\n  binary: %s\n  receipt: %s", result.Ompiglet, result.Version, result.Target, result.SignerKeyID, result.Artifact, result.Receipt)
	if jsonMode {
		writePigletCommandJSON(pigletCommandOutput{Command: "update", Success: true, Output: message}, stdout)
	} else {
		_, _ = fmt.Fprintln(stdout, message)
	}
	return 0
}
