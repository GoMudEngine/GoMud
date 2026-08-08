package playtestenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/version"
)

// runsProbeManifestPath and reportsProbeSentinelPath are representative
// future artifact paths - relative to a selected checkout - that must
// already be Git-ignored before this package will reserve a run. "probe" is
// a literal placeholder, not a real run ID: the check only needs to prove
// the checkout's ignore rules cover the supervisor's whole artifact shape.
const (
	runsProbeManifestPath    = "tools/playtest/.run/probe/manifest.json"
	reportsProbeSentinelPath = "tools/playtest/reports/probe-environment-failed.md"
)

// targetsYAMLExcludedPath is the exact checkout-relative path of the
// existing playtest target-credential file. Its path/status metadata is
// never recorded in a Git baseline, regardless of case, because it can
// contain connection secrets for other environments.
const targetsYAMLExcludedPath = "tools/playtest/targets.yaml"

var (
	// ErrCheckoutNotDirectory is returned when the selected checkout path
	// does not exist, cannot be canonicalized, or is not a directory.
	ErrCheckoutNotDirectory = errors.New("playtestenv: checkout path does not exist or is not a directory")

	// ErrCheckoutArchivePath is returned when the canonical checkout path
	// contains a path component equal to "_archive" (case-insensitively).
	ErrCheckoutArchivePath = errors.New("playtestenv: checkout path contains an _archive path component")

	// ErrCheckoutMissingGoMod is returned when the checkout has no go.mod.
	ErrCheckoutMissingGoMod = errors.New("playtestenv: checkout is missing go.mod")

	// ErrCheckoutMissingDockerfile is returned when the checkout has no
	// provisioning/Dockerfile.
	ErrCheckoutMissingDockerfile = errors.New("playtestenv: checkout is missing provisioning/Dockerfile")

	// ErrCheckoutMissingMain is returned when the checkout has no main.go.
	ErrCheckoutMissingMain = errors.New("playtestenv: checkout is missing main.go")

	// ErrCheckoutNotGitRoot is returned when Git does not identify the
	// selected checkout path itself as a checkout/worktree root.
	ErrCheckoutNotGitRoot = errors.New("playtestenv: checkout is not the root of a git checkout or worktree")

	// ErrCheckoutArtifactsNotIgnored is returned when the checkout's Git
	// ignore rules do not cover this package's future run/report artifact
	// paths.
	ErrCheckoutArtifactsNotIgnored = errors.New("playtestenv: checkout does not ignore the supervisor's run/report artifact paths")

	// ErrCheckoutVersionMissing is returned when main.go does not declare a
	// package-level string-literal constant named VERSION.
	ErrCheckoutVersionMissing = errors.New("playtestenv: checkout main.go does not declare a package-level string-literal VERSION constant")

	// ErrCheckoutVersionInvalid is returned when the declared VERSION
	// constant fails to parse as a valid, non-zero version.
	ErrCheckoutVersionInvalid = errors.New("playtestenv: checkout VERSION constant is not a valid non-zero version")
)

// checkoutValidation is the verified outcome of validating a selected
// checkout: its canonical path, a stable fingerprint derived from that
// path, and the version declared by its main.go.
type checkoutValidation struct {
	Path        string
	Fingerprint string
	Version     version.Version
}

// symlinkResolveFunc matches filepath.EvalSymlinks's signature.
// verifyGitCheckoutRoot takes it as a parameter so tests can exercise the
// Windows/Linux path-comparison logic against a git-reported root that does
// not need to exist on the real filesystem, without depending on real
// symlink resolution or on-disk case normalization.
type symlinkResolveFunc func(string) (string, error)

// validateCheckoutForHost performs the full checkout validation described
// on validateCheckout using the current process's real platform and the
// standard library's real symlink resolution. It is the single production
// entry point.
func validateCheckoutForHost(ctx context.Context, runner Runner, rawPath string) (checkoutValidation, error) {
	return validateCheckout(ctx, runner, rawPath, runtime.GOOS, filepath.EvalSymlinks)
}

// validateCheckout is validateCheckoutForHost with its platform and
// symlink-resolution dependency injected, so unit tests can exercise every
// branch deterministically.
//
// It performs, in order:
//
//  1. Canonicalize rawPath with filepath.Abs, filepath.EvalSymlinks, and
//     filepath.Clean, and require the result to be an existing directory.
//  2. Reject any canonical path component equal to "_archive"
//     case-insensitively.
//  3. Require go.mod, provisioning/Dockerfile, and main.go to exist
//     beneath the canonical path.
//  4. Parse main.go's package-level string-literal VERSION constant and
//     validate it with internal/version.Parse.
//  5. Require `git --no-optional-locks -C <checkout> rev-parse
//     --show-toplevel` to resolve (via filepath.FromSlash,
//     resolveSymlinks, and filepath.Clean) to the same canonical path,
//     compared with strings.EqualFold on Windows and exact equality
//     elsewhere.
//  6. Require `git --no-optional-locks -C <checkout> check-ignore` to
//     succeed for this package's future run-manifest and
//     failed-environment-report paths.
//
// Every Git read uses `git --no-optional-locks -C <checkout> ...` argv, run
// without a shell, so no path can be reinterpreted as a shell token.
func validateCheckout(ctx context.Context, runner Runner, rawPath string, goos string, resolveSymlinks symlinkResolveFunc) (checkoutValidation, error) {
	canonical, err := canonicalizeCheckoutPath(rawPath)
	if err != nil {
		return checkoutValidation{}, fmt.Errorf("%w: %v", ErrCheckoutNotDirectory, err)
	}
	if err := requireDir(canonical); err != nil {
		return checkoutValidation{}, err
	}
	if hasArchivePathComponent(canonical) {
		return checkoutValidation{}, ErrCheckoutArchivePath
	}

	if err := requireFile(filepath.Join(canonical, "go.mod"), ErrCheckoutMissingGoMod); err != nil {
		return checkoutValidation{}, err
	}
	if err := requireFile(filepath.Join(canonical, "provisioning", "Dockerfile"), ErrCheckoutMissingDockerfile); err != nil {
		return checkoutValidation{}, err
	}
	mainGoPath := filepath.Join(canonical, "main.go")
	if err := requireFile(mainGoPath, ErrCheckoutMissingMain); err != nil {
		return checkoutValidation{}, err
	}

	ver, err := parseCheckoutVersion(mainGoPath)
	if err != nil {
		return checkoutValidation{}, err
	}

	if err := verifyGitCheckoutRoot(ctx, runner, canonical, goos, resolveSymlinks); err != nil {
		return checkoutValidation{}, err
	}
	if err := verifyIgnoredArtifactPaths(ctx, runner, canonical); err != nil {
		return checkoutValidation{}, err
	}

	return checkoutValidation{
		Path:        canonical,
		Fingerprint: checkoutFingerprint(canonical, goos),
		Version:     ver,
	}, nil
}

// canonicalizeCheckoutPath resolves rawPath to an absolute, symlink-free,
// cleaned path. It never trusts the caller's spelling, casing, or trailing
// separators.
func canonicalizeCheckoutPath(rawPath string) (string, error) {
	abs, err := filepath.Abs(rawPath)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

// requireDir requires path to exist and be a directory.
func requireDir(path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return ErrCheckoutNotDirectory
	}
	return nil
}

// requireFile requires path to exist and be a regular (non-directory) file,
// returning missingErr otherwise.
func requireFile(path string, missingErr error) error {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return missingErr
	}
	return nil
}

// hasArchivePathComponent reports whether any "/"-delimited component of
// canonical equals "_archive", case-insensitively. It never matches a
// component that merely contains "_archive" as a substring.
func hasArchivePathComponent(canonical string) bool {
	for _, part := range strings.Split(filepath.ToSlash(canonical), "/") {
		if strings.EqualFold(part, "_archive") {
			return true
		}
	}
	return false
}

// checkoutFingerprint returns a stable, hex-encoded SHA-256 digest of
// canonical, normalized deterministically for the target goos. Windows
// paths are lowercased and backslash-normalized to forward slashes even
// when this function runs on a non-Windows host (filepath.ToSlash is
// host-OS-dependent and would leave '\' intact on Linux). Other platforms
// keep case and use filepath.ToSlash.
func checkoutFingerprint(canonical, goos string) string {
	normalized := canonical
	if goos == "windows" {
		normalized = strings.ReplaceAll(normalized, `\`, "/")
		normalized = strings.ToLower(normalized)
	} else {
		normalized = filepath.ToSlash(normalized)
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// parseCheckoutVersion parses mainGoPath with go/parser, requires a
// package-level constant declaration named VERSION whose value is a string
// literal, and validates it with internal/version.Parse. It never uses a
// regular expression or a hard-coded version.
//
// It also honors Go's implicit const-expression repetition: within one
// parenthesized const block, a ConstSpec with no expression list of its own
// inherits the nearest preceding non-empty expression list verbatim (Go
// spec, "Constant declarations"). This is ordinarily seen with iota, but is
// legal for any expression - so `OTHER = "1.2.3"` followed by a bare
// `VERSION` (no `=`) on the next line is a valid string-literal VERSION
// declaration and must resolve the same as `VERSION = "1.2.3"`.
func parseCheckoutVersion(mainGoPath string) (version.Version, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, mainGoPath, nil, 0)
	if err != nil {
		return version.Version{}, fmt.Errorf("%w: parse error: %v", ErrCheckoutVersionMissing, err)
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		var inheritedValues []ast.Expr
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			values := valueSpec.Values
			if len(values) > 0 {
				inheritedValues = values
			} else {
				values = inheritedValues
			}
			for i, name := range valueSpec.Names {
				if name.Name != "VERSION" {
					continue
				}
				if i >= len(values) {
					continue
				}
				lit, ok := values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return version.Version{}, ErrCheckoutVersionMissing
				}
				strVal, err := strconv.Unquote(lit.Value)
				if err != nil {
					return version.Version{}, fmt.Errorf("%w: could not unquote VERSION literal: %v", ErrCheckoutVersionMissing, err)
				}
				ver, err := version.Parse(strVal)
				if err != nil {
					return version.Version{}, fmt.Errorf("%w: %v", ErrCheckoutVersionInvalid, err)
				}
				return ver, nil
			}
		}
	}

	return version.Version{}, ErrCheckoutVersionMissing
}

// verifyGitCheckoutRoot requires `git --no-optional-locks -C <canonical>
// rev-parse --show-toplevel` to report canonical itself as the checkout
// root, after normalizing Git's forward-slash output with
// filepath.FromSlash, resolveSymlinks, and filepath.Clean. Comparison uses
// strings.EqualFold on Windows (whose filesystem paths are
// case-insensitive) and exact equality elsewhere.
func verifyGitCheckoutRoot(ctx context.Context, runner Runner, canonical, goos string, resolveSymlinks symlinkResolveFunc) error {
	out, err := runGitCaptureOutput(ctx, runner, canonical, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("%w: git rev-parse --show-toplevel: %v", ErrCheckoutNotGitRoot, err)
	}

	reported := strings.TrimSpace(out)
	if reported == "" {
		return fmt.Errorf("%w: git rev-parse --show-toplevel returned an empty path", ErrCheckoutNotGitRoot)
	}

	normalized := filepath.Clean(filepath.FromSlash(reported))
	resolved, err := resolveSymlinks(normalized)
	if err != nil {
		return fmt.Errorf("%w: could not resolve reported git root %q: %v", ErrCheckoutNotGitRoot, reported, err)
	}
	resolved = filepath.Clean(resolved)

	matches := resolved == canonical
	if goos == "windows" {
		matches = strings.EqualFold(resolved, canonical)
	}
	if !matches {
		return fmt.Errorf("%w: git reports checkout root %q, expected %q", ErrCheckoutNotGitRoot, resolved, canonical)
	}
	return nil
}

// verifyIgnoredArtifactPaths requires Git to already ignore this package's
// future run-manifest and failed-environment-report paths before a run may
// be reserved beneath the checkout.
func verifyIgnoredArtifactPaths(ctx context.Context, runner Runner, canonical string) error {
	for _, probe := range []string{runsProbeManifestPath, reportsProbeSentinelPath} {
		if _, err := runGitCaptureOutput(ctx, runner, canonical, "check-ignore", "--", probe); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrCheckoutArtifactsNotIgnored, probe, err)
		}
	}
	return nil
}

// gitArgs builds argv beginning "--no-optional-locks -C <checkout>"
// followed by args. --no-optional-locks ensures every read-only Git
// invocation from this package never refreshes or locks the user's index.
func gitArgs(checkout string, args []string) []string {
	full := make([]string, 0, len(args)+3)
	full = append(full, "--no-optional-locks", "-C", checkout)
	full = append(full, args...)
	return full
}

// runGitCaptureOutput runs `git --no-optional-locks -C <checkout>
// <args...>` and returns its captured stdout as a string. On failure, the
// error includes captured stderr.
func runGitCaptureOutput(ctx context.Context, runner Runner, checkout string, args ...string) (string, error) {
	var stdout, stderr strings.Builder
	err := runner.Run(ctx, CommandSpec{
		Name:   "git",
		Args:   gitArgs(checkout, args),
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		if stderrText := strings.TrimSpace(stderr.String()); stderrText != "" {
			return "", fmt.Errorf("%w: %s", err, stderrText)
		}
		return "", err
	}
	return stdout.String(), nil
}

// collectGitBaseline records a non-secret, metadata-only Git baseline of
// checkout: the current commit (if any) from `rev-parse HEAD`, and
// path/status metadata from `status --short -z --untracked-files=all`.
// It never runs `git diff`, never reads file contents, and excludes
// credential, archive, and this package's own artifact paths from the
// recorded entries. All Git path text is treated as untrusted data: it is
// never interpreted by a shell and never used to open a file.
func collectGitBaseline(ctx context.Context, runner Runner, checkout string) (GitBaseline, error) {
	var baseline GitBaseline

	if out, err := runGitCaptureOutput(ctx, runner, checkout, "rev-parse", "HEAD"); err == nil {
		baseline.Commit = strings.TrimSpace(out)
	}

	statusOut, err := runGitCaptureOutput(ctx, runner, checkout, "status", "--short", "-z", "--untracked-files=all")
	if err != nil {
		return GitBaseline{}, fmt.Errorf("playtestenv: git status: %w", err)
	}

	for _, entry := range parseGitStatusZ(statusOut) {
		if gitEntryExcluded(entry) {
			continue
		}
		baseline.Entries = append(baseline.Entries, entry)
	}

	return baseline, nil
}

// parseGitStatusZ parses the NUL-terminated record stream produced by `git
// status --short -z --untracked-files=all`. Each record is "XY PATH\0";
// when X or Y is 'R' (renamed) or 'C' (copied), PATH is followed by one
// additional "\0"-terminated ORIG_PATH record (the source path Git renamed
// or copied from). Hostile path text - embedded spaces, Unicode, newlines,
// or Markdown - is never interpreted: records are split on NUL only, so
// arbitrary bytes within one path field are preserved verbatim as data.
func parseGitStatusZ(raw string) []GitEntry {
	if raw == "" {
		return nil
	}
	tokens := strings.Split(raw, "\x00")
	if len(tokens) > 0 && tokens[len(tokens)-1] == "" {
		tokens = tokens[:len(tokens)-1]
	}

	entries := make([]GitEntry, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if len(tok) < 3 {
			// Malformed/truncated record: skip defensively rather than
			// panic on an index slice.
			continue
		}
		status := tok[:2]
		path := tok[3:]
		entry := GitEntry{Status: status, Path: path}
		if status[0] == 'R' || status[0] == 'C' || status[1] == 'R' || status[1] == 'C' {
			i++
			if i < len(tokens) {
				entry.OrigPath = tokens[i]
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

// isExcludedBaselinePath reports whether p must never appear in a recorded
// Git baseline: the tracked playtest target-credential file, any path
// beneath a component equal to "_archive" (case-insensitively), a basename
// starting with ".env", a basename containing "credential" or "secret", a
// basename ending ".key", or a path beneath this package's own ignored
// run-state or report directories. Every one of these checks is
// case-insensitive: Git path text is untrusted, and a credential file
// renamed with different casing (e.g. on a case-insensitive filesystem, or
// simply by a careless `git mv`) must not silently bypass the filter.
func isExcludedBaselinePath(p string) bool {
	norm := filepath.ToSlash(p)
	if strings.EqualFold(norm, targetsYAMLExcludedPath) {
		return true
	}

	segments := strings.Split(norm, "/")
	for _, seg := range segments {
		if strings.EqualFold(seg, "_archive") {
			return true
		}
	}

	base := segments[len(segments)-1]
	lowerBase := strings.ToLower(base)
	if strings.HasPrefix(lowerBase, ".env") {
		return true
	}
	if strings.Contains(lowerBase, "credential") || strings.Contains(lowerBase, "secret") {
		return true
	}
	if strings.HasSuffix(lowerBase, ".key") {
		return true
	}

	if len(segments) >= 3 && strings.EqualFold(segments[0], "tools") && strings.EqualFold(segments[1], "playtest") &&
		strings.HasPrefix(strings.ToLower(segments[2]), ".run") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(norm), "tools/playtest/reports/") {
		return true
	}

	return false
}

// gitEntryExcluded reports whether e must be dropped from a recorded Git
// baseline because either its current or (for a rename/copy) original path
// matches isExcludedBaselinePath.
func gitEntryExcluded(e GitEntry) bool {
	if isExcludedBaselinePath(e.Path) {
		return true
	}
	if e.OrigPath != "" && isExcludedBaselinePath(e.OrigPath) {
		return true
	}
	return false
}
