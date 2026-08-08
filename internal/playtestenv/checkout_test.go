package playtestenv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeMinimalCheckout creates a minimal, otherwise-valid checkout at dir:
// go.mod, provisioning/Dockerfile, and a main.go declaring the given
// VERSION source snippet (a full package-level const declaration).
func writeMinimalCheckout(t *testing.T, dir string, versionDecl string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "provisioning"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/checkout\n\ngo 1.25\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "provisioning", "Dockerfile"), []byte("FROM alpine\n"), 0o644))
	mainSrc := "package main\n\n" + versionDecl + "\n\nfunc main() {}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainSrc), 0o644))
}

// identitySymlinkResolver returns its input unchanged, letting tests
// exercise the Windows/Linux path-comparison logic against a fabricated
// git-reported root that need not exist on the real filesystem.
func identitySymlinkResolver(p string) (string, error) { return p, nil }

// scriptCheckoutHappyPath registers a fully successful rev-parse and
// check-ignore sequence for canonical on fr.
func scriptCheckoutHappyPath(fr *fakeDockerRunner, canonical string) {
	fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns(canonical+"\n", "", nil)
	fr.script(gitArgs(canonical, []string{"check-ignore", "--", runsProbeManifestPath})...).returns("", "", nil)
	fr.script(gitArgs(canonical, []string{"check-ignore", "--", reportsProbeSentinelPath})...).returns("", "", nil)
}

func TestValidateCheckoutRequiresRepositoryRootGoModAndDockerfile(t *testing.T) {
	t.Run("path does not exist", func(t *testing.T) {
		fr := newFakeDockerRunner()
		missing := filepath.Join(t.TempDir(), "does-not-exist")

		_, err := validateCheckout(context.Background(), fr, missing, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutNotDirectory)
		require.Empty(t, fr.calls)
	})

	t.Run("path is a file, not a directory", func(t *testing.T) {
		fr := newFakeDockerRunner()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "not-a-dir")
		require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))

		_, err := validateCheckout(context.Background(), fr, filePath, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutNotDirectory)
		require.Empty(t, fr.calls)
	})

	t.Run("missing go.mod", func(t *testing.T) {
		fr := newFakeDockerRunner()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "provisioning"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "provisioning", "Dockerfile"), []byte("FROM alpine\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nconst VERSION = \"1.0.0\"\nfunc main() {}\n"), 0o644))

		_, err := validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutMissingGoMod)
		require.Empty(t, fr.calls, "must not invoke git before local filesystem checks pass")
	})

	t.Run("missing provisioning/Dockerfile", func(t *testing.T) {
		fr := newFakeDockerRunner()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nconst VERSION = \"1.0.0\"\nfunc main() {}\n"), 0o644))

		_, err := validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutMissingDockerfile)
		require.Empty(t, fr.calls)
	})

	t.Run("missing main.go", func(t *testing.T) {
		fr := newFakeDockerRunner()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "provisioning"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "provisioning", "Dockerfile"), []byte("FROM alpine\n"), 0o644))

		_, err := validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutMissingMain)
		require.Empty(t, fr.calls)
	})

	t.Run("fully valid checkout succeeds", func(t *testing.T) {
		dir := t.TempDir()
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)
		writeMinimalCheckout(t, dir, `const VERSION = "1.2.3"`)

		fr := newFakeDockerRunner()
		scriptCheckoutHappyPath(fr, canonical)

		got, err := validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.NoError(t, err)
		require.Equal(t, canonical, got.Path)
		require.Equal(t, checkoutFingerprint(canonical, "linux"), got.Fingerprint)
		require.Equal(t, "1.2.3", got.Version.String())
	})
}

func TestValidateCheckoutNormalizesWindowsGitPath(t *testing.T) {
	t.Run("windows tolerates a differently-cased reported root", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		differentCase := strings.ToUpper(canonical)
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns(differentCase+"\n", "", nil)
		fr.script(gitArgs(canonical, []string{"check-ignore", "--", runsProbeManifestPath})...).returns("", "", nil)
		fr.script(gitArgs(canonical, []string{"check-ignore", "--", reportsProbeSentinelPath})...).returns("", "", nil)

		_, err = validateCheckout(context.Background(), fr, dir, "windows", identitySymlinkResolver)

		require.NoError(t, err, "windows comparison must be case-insensitive")
	})

	t.Run("linux rejects the same case difference", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		differentCase := strings.ToUpper(canonical)
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns(differentCase+"\n", "", nil)

		_, err = validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutNotGitRoot, "linux comparison must be case-sensitive")
	})

	t.Run("forward-slash git output is normalized via FromSlash", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		forwardSlashForm := filepath.ToSlash(canonical)
		fr := newFakeDockerRunner()
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns(forwardSlashForm+"\n", "", nil)
		fr.script(gitArgs(canonical, []string{"check-ignore", "--", runsProbeManifestPath})...).returns("", "", nil)
		fr.script(gitArgs(canonical, []string{"check-ignore", "--", reportsProbeSentinelPath})...).returns("", "", nil)

		_, err = validateCheckout(context.Background(), fr, dir, "windows", identitySymlinkResolver)

		require.NoError(t, err)
	})

	t.Run("mismatched root is rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns(filepath.Dir(canonical)+"\n", "", nil)

		_, err = validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutNotGitRoot)
	})

	t.Run("empty reported root is rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns("\n", "", nil)

		_, err = validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutNotGitRoot)
	})

	t.Run("rev-parse failure is rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns("", "fatal: not a git repository", errors.New("exit status 128"))

		_, err = validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutNotGitRoot)
	})

	t.Run("unresolvable reported root is rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns(canonical+"\n", "", nil)
		failingResolver := func(string) (string, error) { return "", errors.New("no such path") }

		_, err = validateCheckout(context.Background(), fr, dir, "linux", failingResolver)

		require.ErrorIs(t, err, ErrCheckoutNotGitRoot)
	})
}

func TestValidateCheckoutRejectsArchivePathComponent(t *testing.T) {
	cases := []string{"_archive", "_ARCHIVE", "_Archive"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			nested := filepath.Join(root, name, "repo")
			require.NoError(t, os.MkdirAll(nested, 0o755))
			writeMinimalCheckout(t, nested, `const VERSION = "1.0.0"`)

			fr := newFakeDockerRunner()
			_, err := validateCheckout(context.Background(), fr, nested, "linux", identitySymlinkResolver)

			require.ErrorIs(t, err, ErrCheckoutArchivePath)
			require.Empty(t, fr.calls, "must not invoke git once an _archive component is found")
		})
	}

	t.Run("substring match does not trigger rejection", func(t *testing.T) {
		root := t.TempDir()
		nested := filepath.Join(root, "my_archive_stuff", "repo")
		require.NoError(t, os.MkdirAll(nested, 0o755))
		writeMinimalCheckout(t, nested, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(nested)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		scriptCheckoutHappyPath(fr, canonical)

		_, err = validateCheckout(context.Background(), fr, nested, "linux", identitySymlinkResolver)

		require.NoError(t, err, "a component that only contains _archive as a substring must not be rejected")
	})
}

func TestValidateCheckoutRequiresIgnoredRunAndReportPaths(t *testing.T) {
	t.Run("manifest path not ignored", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns(canonical+"\n", "", nil)
		fr.script(gitArgs(canonical, []string{"check-ignore", "--", runsProbeManifestPath})...).returns("", "", errors.New("exit status 1"))

		_, err = validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutArtifactsNotIgnored)
	})

	t.Run("report path not ignored", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		fr.script(gitArgs(canonical, []string{"rev-parse", "--show-toplevel"})...).returns(canonical+"\n", "", nil)
		fr.script(gitArgs(canonical, []string{"check-ignore", "--", runsProbeManifestPath})...).returns("", "", nil)
		fr.script(gitArgs(canonical, []string{"check-ignore", "--", reportsProbeSentinelPath})...).returns("", "", errors.New("exit status 1"))

		_, err = validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.ErrorIs(t, err, ErrCheckoutArtifactsNotIgnored)
	})

	t.Run("both ignored succeeds", func(t *testing.T) {
		dir := t.TempDir()
		writeMinimalCheckout(t, dir, `const VERSION = "1.0.0"`)
		canonical, err := canonicalizeCheckoutPath(dir)
		require.NoError(t, err)

		fr := newFakeDockerRunner()
		scriptCheckoutHappyPath(fr, canonical)

		_, err = validateCheckout(context.Background(), fr, dir, "linux", identitySymlinkResolver)

		require.NoError(t, err)
		for _, call := range fr.calls {
			require.Equal(t, "git", call.Name)
		}
	})
}

func TestCheckoutFingerprintIsStableForCanonicalPath(t *testing.T) {
	t.Run("windows folds case", func(t *testing.T) {
		a := checkoutFingerprint(`C:\Users\Foo\Bar`, "windows")
		b := checkoutFingerprint(`c:\users\foo\bar`, "windows")
		require.Equal(t, a, b)
		require.Len(t, a, 64, "must be a hex-encoded SHA-256 digest")
	})

	t.Run("windows normalizes slash direction", func(t *testing.T) {
		a := checkoutFingerprint(`C:\Users\Foo`, "windows")
		b := checkoutFingerprint(`C:/Users/Foo`, "windows")
		require.Equal(t, a, b)
	})

	t.Run("linux is case-sensitive", func(t *testing.T) {
		a := checkoutFingerprint("/home/foo/Bar", "linux")
		b := checkoutFingerprint("/home/foo/bar", "linux")
		require.NotEqual(t, a, b)
	})

	t.Run("same input is always stable", func(t *testing.T) {
		p := `C:\Users\Foo\Bar`
		require.Equal(t, checkoutFingerprint(p, "windows"), checkoutFingerprint(p, "windows"))
	})

	t.Run("distinct paths produce distinct fingerprints", func(t *testing.T) {
		require.NotEqual(t, checkoutFingerprint("/one", "linux"), checkoutFingerprint("/two", "linux"))
	})
}

func TestCheckoutVersionRequiresValidNonzeroStringLiteralVERSION(t *testing.T) {
	cases := []struct {
		name       string
		mainGoBody string
		wantValid  bool
		wantErr    error
		wantString string
	}{
		{
			name:       "valid version",
			mainGoBody: `const VERSION = "1.2.3"`,
			wantValid:  true,
			wantString: "1.2.3",
		},
		{
			name:       "missing VERSION constant entirely",
			mainGoBody: `const OTHER = "1.2.3"`,
			wantErr:    ErrCheckoutVersionMissing,
		},
		{
			name:       "VERSION is not a string literal",
			mainGoBody: `const VERSION = 5`,
			wantErr:    ErrCheckoutVersionMissing,
		},
		{
			name:       "VERSION is a computed expression",
			mainGoBody: "func computeVersion() string { return \"1.0.0\" }\n\nvar VERSION = computeVersion()",
			wantErr:    ErrCheckoutVersionMissing,
		},
		{
			name:       "VERSION does not parse",
			mainGoBody: `const VERSION = "not-a-version"`,
			wantErr:    ErrCheckoutVersionInvalid,
		},
		{
			name:       "VERSION is the zero version",
			mainGoBody: `const VERSION = "0.0.0"`,
			wantErr:    ErrCheckoutVersionInvalid,
		},
		{
			name:       "lowercase version name is ignored",
			mainGoBody: `const version = "1.2.3"`,
			wantErr:    ErrCheckoutVersionMissing,
		},
		{
			name:       "VERSION declared inside a function is invisible",
			mainGoBody: "func init() { const VERSION = \"1.2.3\"; _ = VERSION }",
			wantErr:    ErrCheckoutVersionMissing,
		},
		{
			name:       "VERSION among multiple consts in one block",
			mainGoBody: "const (\n\tOTHER = \"x\"\n\tVERSION = \"2.5.0\"\n)",
			wantValid:  true,
			wantString: "2.5.0",
		},
		{
			// Legal Go: within a parenthesized const block, a ConstSpec
			// with no expression list inherits the nearest preceding
			// non-empty expression list (and type, if any) verbatim. This
			// is normally seen with iota, but is legal for any expression,
			// including a bare string literal - VERSION here has no `=`
			// of its own and must still resolve to OTHER's "1.2.3".
			name:       "implicit const expression repetition inherits string VERSION",
			mainGoBody: "const (\n\tOTHER = \"1.2.3\"\n\tVERSION\n)",
			wantValid:  true,
			wantString: "1.2.3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mainSrc := "package main\n\n" + tc.mainGoBody + "\n\nfunc main() {}\n"
			path := filepath.Join(dir, "main.go")
			require.NoError(t, os.WriteFile(path, []byte(mainSrc), 0o644))

			ver, err := parseCheckoutVersion(path)

			if tc.wantValid {
				require.NoError(t, err)
				require.Equal(t, tc.wantString, ver.String())
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestGitBaselineStoresOnlyPathAndStatusMetadata(t *testing.T) {
	dir := t.TempDir()
	fr := newFakeDockerRunner()
	fr.script(gitArgs(dir, []string{"rev-parse", "HEAD"})...).returns("deadbeefcafef00d\n", "", nil)

	tokens := []string{
		"M  normal/modified.txt",
		"?? weird path with spaces and \u65e5\u672c\u8a9e.txt",
		"?? file\nwith\nembedded\nnewlines *markdown* [link](evil).md",
		"R  new_name.txt",
		"old_name.txt",
	}
	raw := strings.Join(tokens, "\x00") + "\x00"
	fr.script(gitArgs(dir, []string{"status", "--short", "-z", "--untracked-files=all"})...).returns(raw, "", nil)

	baseline, err := collectGitBaseline(context.Background(), fr, dir)

	require.NoError(t, err)
	require.Equal(t, "deadbeefcafef00d", baseline.Commit)
	require.Len(t, baseline.Entries, 4)

	require.Equal(t, GitEntry{Status: "M ", Path: "normal/modified.txt"}, baseline.Entries[0])
	require.Equal(t, "??", baseline.Entries[1].Status)
	require.Equal(t, "weird path with spaces and \u65e5\u672c\u8a9e.txt", baseline.Entries[1].Path)
	require.Equal(t, "??", baseline.Entries[2].Status)
	require.Equal(t, "file\nwith\nembedded\nnewlines *markdown* [link](evil).md", baseline.Entries[2].Path)
	require.Equal(t, "R ", baseline.Entries[3].Status)
	require.Equal(t, "new_name.txt", baseline.Entries[3].Path)
	require.Equal(t, "old_name.txt", baseline.Entries[3].OrigPath)

	// GitEntry carries only Status/Path/OrigPath - no other metadata leaks.
	for _, e := range baseline.Entries {
		require.NotContains(t, e.Path, "\x00")
	}
}

func TestGitBaselineStoresHostilePathTextVerbatim(t *testing.T) {
	dir := t.TempDir()
	fr := newFakeDockerRunner()
	fr.script(gitArgs(dir, []string{"rev-parse", "HEAD"})...).returns("", "", errors.New("fatal: bad revision 'HEAD'"))

	spacedPath := "weird path with spaces and \u65e5\u672c\u8a9e.txt"
	markdownPath := "file\nwith\nembedded\nnewlines *markdown* [link](evil).md"
	tokens := []string{
		"?? " + spacedPath,
		"?? " + markdownPath,
		"R  new_name.txt",
		"old_name.txt",
	}
	raw := strings.Join(tokens, "\x00") + "\x00"
	fr.script(gitArgs(dir, []string{"status", "--short", "-z", "--untracked-files=all"})...).returns(raw, "", nil)

	baseline, err := collectGitBaseline(context.Background(), fr, dir)

	require.NoError(t, err)
	require.Empty(t, baseline.Commit, "a rev-parse HEAD failure (e.g. no commits yet) must not fail baseline collection")
	require.Len(t, baseline.Entries, 3)
	require.Equal(t, spacedPath, baseline.Entries[0].Path)
	require.Equal(t, markdownPath, baseline.Entries[1].Path)
	require.Equal(t, "new_name.txt", baseline.Entries[2].Path)
	require.Equal(t, "old_name.txt", baseline.Entries[2].OrigPath)
}

func TestGitBaselineExcludesCredentialsArchiveAndSupervisorArtifacts(t *testing.T) {
	dir := t.TempDir()
	fr := newFakeDockerRunner()
	fr.script(gitArgs(dir, []string{"rev-parse", "HEAD"})...).returns("abc123\n", "", nil)

	tokens := []string{
		"M  tools/playtest/targets.yaml",
		"M  _archive/old_notes.txt",
		"M  nested/_ARCHIVE/deep/file.txt",
		"?? .env",
		"?? .env.local",
		"?? config/MY_CREDENTIAL.txt",
		"?? config/some-secret-value.cfg",
		"?? keys/id_rsa.key",
		"?? tools/playtest/.run/abc123/manifest.json",
		"?? tools/playtest/.run-old/leftover.log",
		"?? tools/playtest/reports/2026-08-07-abc-environment-failed.md",
		"R  tools/playtest/targets.yaml",
		"legit/renamed_from.txt",
		"R  legit/renamed_to.txt",
		"tools/playtest/targets.yaml",
		"M  legitimate/tracked_file.go",
		"?? legitimate/new_file.go",
	}
	raw := strings.Join(tokens, "\x00") + "\x00"
	fr.script(gitArgs(dir, []string{"status", "--short", "-z", "--untracked-files=all"})...).returns(raw, "", nil)

	baseline, err := collectGitBaseline(context.Background(), fr, dir)

	require.NoError(t, err)
	require.Len(t, baseline.Entries, 2, "only the two legitimate entries must survive filtering")
	paths := []string{baseline.Entries[0].Path, baseline.Entries[1].Path}
	require.ElementsMatch(t, []string{"legitimate/tracked_file.go", "legitimate/new_file.go"}, paths)
}

// TestGitBaselineExcludesCredentialsCaseInsensitively targets the three
// checks that must fold case but historically compared against the
// original-case basename/path instead of a lowercased form: the exact
// targets.yaml path match, the ".env" prefix check, and the ".key" suffix
// check. It also proves a rename's OrigPath is excluded case-insensitively:
// a case-variant credential path hidden behind a harmless-looking new name
// must still drop the whole entry.
func TestGitBaselineExcludesCredentialsCaseInsensitively(t *testing.T) {
	dir := t.TempDir()
	fr := newFakeDockerRunner()
	fr.script(gitArgs(dir, []string{"rev-parse", "HEAD"})...).returns("abc123\n", "", nil)

	tokens := []string{
		"M  Tools/Playtest/Targets.YAML", // exact-path match must fold case
		"?? config/.ENV.production",      // ".env" prefix match must fold case
		"?? keys/ID_RSA.KEY",             // ".key" suffix match must fold case
		"R  legit/renamed_to.txt",        // renamed TO a harmless-looking name...
		"Tools/Playtest/Targets.yaml",    // ...FROM a case-variant credential path
		"M  legitimate/tracked_file.go",
	}
	raw := strings.Join(tokens, "\x00") + "\x00"
	fr.script(gitArgs(dir, []string{"status", "--short", "-z", "--untracked-files=all"})...).returns(raw, "", nil)

	baseline, err := collectGitBaseline(context.Background(), fr, dir)

	require.NoError(t, err)
	require.Len(t, baseline.Entries, 1, "every case-variant credential path, including a rename's OrigPath, must still be excluded")
	require.Equal(t, "legitimate/tracked_file.go", baseline.Entries[0].Path)
}

func TestGitBaselineNeverInvokesDiffOrReadsFiles(t *testing.T) {
	dir := t.TempDir()
	// Create a real file whose content would poison the baseline if this
	// package ever opened it - collectGitBaseline must never do so. The
	// name deliberately avoids this package's own credential-style filters
	// so the test proves "content is never read", not "path is filtered".
	poisonPath := "content-should-never-be-opened.txt"
	require.NoError(t, os.WriteFile(filepath.Join(dir, poisonPath), []byte("PASSWORD=hunter2"), 0o644))

	fr := newFakeDockerRunner()
	fr.script(gitArgs(dir, []string{"rev-parse", "HEAD"})...).returns("abc123\n", "", nil)
	fr.script(gitArgs(dir, []string{"status", "--short", "-z", "--untracked-files=all"})...).
		returns("?? "+poisonPath+"\x00", "", nil)

	baseline, err := collectGitBaseline(context.Background(), fr, dir)

	require.NoError(t, err)
	require.Len(t, fr.calls, 2, "must issue exactly rev-parse HEAD and status, nothing else")
	for _, call := range fr.calls {
		require.Equal(t, "git", call.Name)
		for _, arg := range call.Args {
			require.NotEqual(t, "diff", arg, "must never invoke git diff")
		}
	}
	require.Len(t, baseline.Entries, 1)
	require.Equal(t, poisonPath, baseline.Entries[0].Path)
	require.NotContains(t, baseline.Commit, "hunter2")
}
