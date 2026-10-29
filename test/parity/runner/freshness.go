//go:build parity

package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// FreshnessReport describes whether the ompig binary under test was built after
// the source files that feed cmd/ompig. It exists to prevent the most expensive
// parity failure mode: comparing upstream pi against a stale installed ompig while
// editing source in this checkout.
type FreshnessReport struct {
	BinaryPath     string
	BinaryModTime  time.Time
	NewestPath     string
	NewestModTime  time.Time
	Fresh          bool
	CheckedSources []string
}

// CheckPigFreshness compares ompigBin's mtime with the newest Go source file in
// the production packages compiled into the ompig binary. Non-source docs,
// parity scenarios, examples, and testdata are intentionally ignored.
func CheckPigFreshness(repoRoot, ompigBin string) (FreshnessReport, error) {
	report := FreshnessReport{BinaryPath: ompigBin}
	info, err := os.Stat(ompigBin)
	if err != nil {
		return report, fmt.Errorf("stat ompig binary %s: %w", ompigBin, err)
	}
	report.BinaryModTime = info.ModTime()

	for _, rel := range ompigFreshnessSourceDirs {
		root := filepath.Join(repoRoot, rel)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		report.CheckedSources = append(report.CheckedSources, rel)
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				base := d.Name()
				if base == "testdata" || base == "target" || base == "__pycache__" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			if fi.ModTime().After(report.NewestModTime) {
				report.NewestModTime = fi.ModTime()
				report.NewestPath = path
			}
			return nil
		}); err != nil {
			return report, err
		}
	}

	// Allow a one-second filesystem timestamp precision margin.
	report.Fresh = report.NewestPath == "" || !report.NewestModTime.After(report.BinaryModTime.Add(time.Second))
	return report, nil
}

var ompigFreshnessSourceDirs = []string{
	"cmd/ompig",
	"agent",
	"ai",
	"coding",
	"internal",
	"ompiglets/standard",
}

func requireFreshPig(t *testing.T, repoRoot string, ompig BinaryRef, allowStale bool) FreshnessReport {
	t.Helper()
	report, err := CheckPigFreshness(repoRoot, ompig.Path)
	if err != nil {
		t.Fatalf("ompig freshness check failed: %v", err)
	}
	if !report.Fresh && !allowStale {
		t.Fatalf("stale ompig binary under test: %s\n  binary mtime: %s\n  newest source: %s\n  source mtime: %s\n\nBuild a fresh parity binary and rerun:\n  go build -o /tmp/ompig-under-test ./cmd/ompig\n  OMPIG_PARITY_OMPIG_BIN=/tmp/ompig-under-test go test -tags=parity ./test/parity/runner ...\n\nOr use make qc / make parity, which build a fresh bin/ompig-parity by default. To inspect an intentionally stale installed binary, pass -ompig-parity.allow-stale=true.",
			report.BinaryPath,
			report.BinaryModTime.Format(time.RFC3339Nano),
			report.NewestPath,
			report.NewestModTime.Format(time.RFC3339Nano))
	}
	return report
}
