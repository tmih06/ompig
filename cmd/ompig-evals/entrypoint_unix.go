//go:build unix

package main

import (
	"context"
	"os"

	"github.com/tmih06/ompig/internal/evals"
	"github.com/tmih06/ompig/internal/evals/evalsuites"
)

// Paths of the evaluator image (internal/evals/docker/Dockerfile).
const imageRoot = "/opt/ompig-evals"

func entrypoint(ctx context.Context, args []string) (int, error) {
	config := evals.EntrypointConfig{
		PackageRoot: "/repo/packages/evals", Runner: imageRoot + "/ompig-evals",
		Binaries:   []string{imageRoot + "/ompig", imageRoot + "/ompig-eval-extension"},
		AuthSource: "/run/pi-eval-secrets/auth.json", HostAgentDir: "/tmp/pi-eval-host-agent", ArtifactDir: "/artifacts",
		Suites: evals.Runner{Suites: evalsuites.All(), PackageRoot: "/repo/packages/evals"},
	}
	return evals.RunEntrypoint(ctx, config, args, os.Stdout)
}
