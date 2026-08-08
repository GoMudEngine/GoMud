package playtestenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFailureReportCoversEveryCategoryAndOmitsSecrets(t *testing.T) {
	categories := []FailureCategory{
		FailureInvalidCheckout,
		FailureDockerUnavailable,
		FailureBuild,
		FailureContainerExited,
		FailureBootPanic,
		FailureListenerCreation,
		FailurePortPublication,
		FailureNonLoopback,
		FailureReadinessTimeout,
		FailureConnectionProbe,
		FailureManifest,
		FailureCleanup,
		FailureLockBusy,
		FailureAbandonedRun,
	}

	for _, cat := range categories {
		t.Run(string(cat), func(t *testing.T) {
			checkout := t.TempDir()
			reportsDir := filepath.Join(checkout, "tools", "playtest", "reports")
			require.NoError(t, os.MkdirAll(reportsDir, 0o755))

			runDir := filepath.Join(checkout, "tools", "playtest", ".run", "run-report")
			require.NoError(t, os.MkdirAll(filepath.Join(runDir, "control"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(runDir, "build.log"), []byte("build ok\n"), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(runDir, "server.log"), []byte("Server Ready\n"), 0o644))

			m := &Manifest{
				RunID:    "run-report",
				Project:  "dogmud-playtest-run-report",
				Checkout: checkout,
				State:    StateFailed,
				Failure: &FailureRecord{
					Category:  cat,
					Phase:     StateStarting,
					Summary:   "example failure for " + string(cat),
					Retryable: cat == FailureLockBusy,
				},
				Artifacts: ArtifactPaths{
					Manifest:  filepath.Join(runDir, "manifest.json"),
					BuildLog:  filepath.Join(runDir, "build.log"),
					ServerLog: filepath.Join(runDir, "server.log"),
					Compose:   filepath.Join(runDir, "compose.resolved.yml"),
					Config:    filepath.Join(runDir, "control", "config-overrides.yaml"),
				},
			}
			cleanup := &CleanupResult{Complete: true, Summary: "resources removed"}

			now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
			path, err := writeFailureReport(checkout, m, cleanup, now)
			require.NoError(t, err)
			require.FileExists(t, path)
			require.Contains(t, filepath.Base(path), "run-report")
			require.True(t, strings.HasSuffix(path, "-environment-failed.md"))

			body, err := os.ReadFile(path)
			require.NoError(t, err)
			text := string(body)

			require.Contains(t, text, checkout)
			require.Contains(t, text, "run-report")
			require.Contains(t, text, string(StateStarting))
			require.Contains(t, text, string(cat))
			require.Contains(t, text, m.Artifacts.BuildLog)
			require.Contains(t, text, m.Artifacts.ServerLog)
			require.Contains(t, text, "resources removed")

			// Must never leak secrets, config bodies, production endpoints, or diffs.
			for _, banned := range []string{
				"password",
				"PASSWORD",
				"AIPort:",
				"CurrentVersion:",
				"diff --git",
				"@@",
				"targets.yaml",
				"prod.example.com",
				"secret-token",
			} {
				require.NotContains(t, strings.ToLower(text), strings.ToLower(banned))
			}
		})
	}
}

func TestFailureReportHandlesHostileMarkdownPathsAndCollisions(t *testing.T) {
	checkout := t.TempDir()
	reportsDir := filepath.Join(checkout, "tools", "playtest", "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0o755))

	hostileCheckout := checkout + string(filepath.Separator) + "evil`*](javascript:alert(1))"
	// Use a path that exists for writing under checkout; inject hostile text via
	// manifest fields that are rendered as data.
	runID := "run-`drop`"
	m := &Manifest{
		RunID:    runID,
		Project:  "dogmud-playtest-" + runID,
		Checkout: hostileCheckout,
		State:    StateFailed,
		Failure: &FailureRecord{
			Category: FailureBuild,
			Phase:    StateBuilding,
			Summary:  "build failed with <script>alert(1)</script> and [click](http://evil)",
		},
		Artifacts: ArtifactPaths{
			Manifest:  filepath.Join(checkout, "tools", "playtest", ".run", "x", "manifest.json"),
			BuildLog:  filepath.Join(checkout, "tools", "playtest", ".run", "x", "build.log"),
			ServerLog: filepath.Join(checkout, "tools", "playtest", ".run", "x", "server.log"),
			Compose:   filepath.Join(checkout, "tools", "playtest", ".run", "x", "compose.resolved.yml"),
			Config:    filepath.Join(checkout, "tools", "playtest", ".run", "x", "control", "config-overrides.yaml"),
		},
	}

	now := time.Date(2026, 8, 8, 13, 0, 0, 0, time.UTC)
	path1, err := writeFailureReport(checkout, m, &CleanupResult{Complete: false, Summary: "leftover container `abc`"}, now)
	require.NoError(t, err)

	path2, err := writeFailureReport(checkout, m, &CleanupResult{Complete: true, Summary: "cleaned"}, now)
	require.NoError(t, err)
	require.NotEqual(t, path1, path2, "collision must produce a distinct report path")
	require.FileExists(t, path1)
	require.FileExists(t, path2)

	body, err := os.ReadFile(path1)
	require.NoError(t, err)
	text := string(body)
	// Hostile markup must appear as escaped/literal data inside a code span.
	require.Contains(t, text, "build failed")
	require.NotContains(t, text, "<script>")
	require.Contains(t, text, "&lt;script&gt;")
	// The summary is wrapped in a single code span, so a raw Markdown link
	// token sequence must not appear outside backticks as a live link.
	require.Contains(t, text, "[click](http://evil)")
	require.Regexp(t, "`[^`]*\\[click\\]\\(http://evil\\)[^`]*`", text)
}

func TestFailureReportToleratesMissingLogs(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(checkout, "tools", "playtest", "reports"), 0o755))

	m := &Manifest{
		RunID:    "run-missing-logs",
		Project:  "dogmud-playtest-run-missing-logs",
		Checkout: checkout,
		State:    StateFailed,
		Failure: &FailureRecord{
			Category: FailureDockerUnavailable,
			Phase:    StateValidating,
			Summary:  "docker unavailable before build",
		},
		Artifacts: ArtifactPaths{
			Manifest:  filepath.Join(checkout, "tools", "playtest", ".run", "run-missing-logs", "manifest.json"),
			BuildLog:  filepath.Join(checkout, "tools", "playtest", ".run", "run-missing-logs", "build.log"),
			ServerLog: filepath.Join(checkout, "tools", "playtest", ".run", "run-missing-logs", "server.log"),
		},
	}

	path, err := writeFailureReport(checkout, m, &CleanupResult{Complete: true, Summary: "nothing to remove"}, time.Date(2026, 8, 8, 14, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(body)
	require.Contains(t, text, "missing")
	require.Contains(t, text, "build.log")
	require.Contains(t, text, "server.log")
}
