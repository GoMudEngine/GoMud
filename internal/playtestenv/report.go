package playtestenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	reportsDirName          = "tools/playtest/reports"
	environmentFailedSuffix = "-environment-failed.md"
)

// writeFailureReport writes a non-secret Markdown failure report under
// <checkout>/tools/playtest/reports/<timestamp>-<run-id>-environment-failed.md.
// Hostile Markdown/path characters in data fields are escaped so they cannot
// introduce links or HTML. On name collision, a numeric suffix is appended
// before the extension.
func writeFailureReport(checkout string, m *Manifest, cleanup *CleanupResult, now time.Time) (string, error) {
	if checkout == "" || m == nil {
		return "", fmt.Errorf("playtestenv: failure report requires checkout and manifest")
	}
	reportsDir := filepath.Join(checkout, filepath.FromSlash(reportsDirName))
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		return "", fmt.Errorf("playtestenv: create reports directory: %w", err)
	}

	stamp := now.UTC().Format("20060102T150405Z")
	base := stamp + "-" + sanitizeReportToken(m.RunID) + environmentFailedSuffix
	path, err := uniqueReportPath(reportsDir, base)
	if err != nil {
		return "", err
	}

	phase := m.State
	category := FailureCategory("")
	summary := ""
	if m.Failure != nil {
		phase = m.Failure.Phase
		category = m.Failure.Category
		summary = m.Failure.Summary
	}

	var b strings.Builder
	b.WriteString("# Playtest environment failure\n\n")
	b.WriteString("- Checkout: ")
	b.WriteString(escapeReportData(m.Checkout))
	b.WriteString("\n")
	b.WriteString("- Run ID: ")
	b.WriteString(escapeReportData(m.RunID))
	b.WriteString("\n")
	b.WriteString("- Phase: ")
	b.WriteString(escapeReportData(string(phase)))
	b.WriteString("\n")
	b.WriteString("- Category: ")
	b.WriteString(escapeReportData(string(category)))
	b.WriteString("\n")
	if summary != "" {
		b.WriteString("- Summary: ")
		b.WriteString(escapeReportData(summary))
		b.WriteString("\n")
	}
	b.WriteString("\n## Artifacts\n\n")
	writeArtifactLine(&b, "Manifest", m.Artifacts.Manifest)
	writeArtifactLine(&b, "Build log", m.Artifacts.BuildLog)
	writeArtifactLine(&b, "Server log", m.Artifacts.ServerLog)
	writeArtifactLine(&b, "Inspect", m.Artifacts.Inspect)
	writeArtifactLine(&b, "Compose", m.Artifacts.Compose)
	writeArtifactLine(&b, "Config", m.Artifacts.Config)
	// Creds path only — never open or embed creds.json body (passwords).
	writeArtifactLine(&b, "Creds", m.Artifacts.Creds)

	b.WriteString("\n## Log availability\n\n")
	b.WriteString(logAvailabilityLine("Build log", m.Artifacts.BuildLog))
	b.WriteString(logAvailabilityLine("Server log", m.Artifacts.ServerLog))

	b.WriteString("\n## Cleanup\n\n")
	if cleanup == nil {
		b.WriteString("- Result: not attempted\n")
	} else {
		b.WriteString("- Complete: ")
		b.WriteString(fmt.Sprintf("%t", cleanup.Complete))
		b.WriteString("\n")
		if cleanup.Summary != "" {
			b.WriteString("- Summary: ")
			b.WriteString(escapeReportData(cleanup.Summary))
			b.WriteString("\n")
		}
		for _, left := range cleanup.Leftovers {
			b.WriteString("- Leftover: ")
			b.WriteString(escapeReportData(left.Kind))
			b.WriteString(" ")
			b.WriteString(escapeReportData(left.ID))
			b.WriteString("\n")
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("playtestenv: write failure report: %w", err)
	}
	return path, nil
}

func writeArtifactLine(b *strings.Builder, label, path string) {
	b.WriteString("- ")
	b.WriteString(label)
	b.WriteString(": ")
	if path == "" {
		b.WriteString("(none)")
	} else {
		b.WriteString(escapeReportData(path))
	}
	b.WriteString("\n")
}

func logAvailabilityLine(label, path string) string {
	if path == "" {
		return fmt.Sprintf("- %s: missing\n", label)
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Sprintf("- %s: missing (%s)\n", label, escapeReportData(path))
	}
	return fmt.Sprintf("- %s: present (%s)\n", label, escapeReportData(path))
}

// escapeReportData renders s as a Markdown code span so hostile path or
// summary text cannot introduce links, images, or raw HTML. Backticks inside
// the value are neutralized so they cannot break out of the span; other
// characters (including underscores in paths and failure categories) are
// preserved literally inside the code span.
func escapeReportData(s string) string {
	safe := strings.ReplaceAll(s, "`", "'")
	safe = strings.ReplaceAll(safe, "<", "&lt;")
	safe = strings.ReplaceAll(safe, ">", "&gt;")
	return "`" + safe + "`"
}

func sanitizeReportToken(runID string) string {
	if runID == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range runID {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := b.String()
	if out == "" {
		return "unknown"
	}
	return out
}

func uniqueReportPath(reportsDir, base string) (string, error) {
	candidate := filepath.Join(reportsDir, base)
	if _, err := os.Stat(candidate); err != nil {
		if os.IsNotExist(err) {
			return candidate, nil
		}
		return "", fmt.Errorf("playtestenv: stat report path: %w", err)
	}
	stem := strings.TrimSuffix(base, environmentFailedSuffix)
	for i := 2; i < 1000; i++ {
		alt := filepath.Join(reportsDir, fmt.Sprintf("%s-%d%s", stem, i, environmentFailedSuffix))
		if _, err := os.Stat(alt); err != nil {
			if os.IsNotExist(err) {
				return alt, nil
			}
			return "", fmt.Errorf("playtestenv: stat report path: %w", err)
		}
	}
	return "", fmt.Errorf("playtestenv: exhausted report name collision retries")
}
