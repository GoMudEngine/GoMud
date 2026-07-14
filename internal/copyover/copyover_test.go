//go:build !windows

package copyover_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain intercepts subprocess invocations. When GO_COPYOVER_HELPER is set,
// this process acts as the "restored" child: it reads the pipe fd, restores
// state from all registered contributors, then writes the restored state as
// JSON to stdout for the parent test to assert against.
func TestMain(m *testing.M) {
	if os.Getenv("GO_COPYOVER_HELPER") != "1" {
		os.Exit(m.Run())
	}

	fdStr := os.Getenv("GO_COPYOVER_FD")
	fd, err := strconv.Atoi(fdStr)
	if err != nil || fd < 0 {
		fmt.Fprintf(os.Stderr, "copyover helper: invalid GO_COPYOVER_FD %q\n", fdStr)
		os.Exit(1)
	}

	alpha := &captureContributor{name: "alpha"}
	beta := &captureContributor{name: "beta"}
	copyover.Register(alpha)
	copyover.Register(beta)

	if err := copyover.Restore(fd); err != nil {
		fmt.Fprintf(os.Stderr, "copyover helper: Restore: %v\n", err)
		os.Exit(1)
	}

	out := map[string]string{
		"alpha": alpha.restored,
		"beta":  beta.restored,
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "copyover helper: encode output: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

// captureContributor is a test-only Contributor that saves a fixed string
// payload and records whatever string it receives during Restore.
type captureContributor struct {
	name     string
	payload  string
	restored string
}

func (c *captureContributor) CopyoverName() string { return c.name }

func (c *captureContributor) CopyoverSave(enc *copyover.Encoder) error {
	return enc.WriteSection(c.name, c.payload)
}

func (c *captureContributor) CopyoverRestore(dec *copyover.Decoder) error {
	var v string
	if err := dec.ReadSection(c.name, &v); err != nil {
		return err
	}
	c.restored = v
	return nil
}

// TestEncoderDecoder_Roundtrip verifies that the wire format preserves values
// exactly across multiple sections, including non-ASCII content. It goes through
// Save/Restore since the encoder and decoder are only well-formed as a pair
// produced by Save (which writes the section count header).
func TestEncoderDecoder_Roundtrip(t *testing.T) {
	type payload struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	cases := []struct {
		name string
		data payload
	}{
		{"section-a", payload{"hello", 1}},
		{"section-b", payload{"world", 42}},
		{"unicode", payload{"日本語テスト", 0}},
	}

	// Build contributors that hold the expected payloads.
	type structContributor struct {
		name     string
		data     payload
		restored payload
	}
	contribs := make([]*structContributor, len(cases))
	for i, tc := range cases {
		contribs[i] = &structContributor{name: tc.name, data: tc.data}
	}

	copyover.ResetRegistry()
	for _, c := range contribs {
		c := c
		copyover.Register(copyover.FuncContributor(
			c.name,
			func(enc *copyover.Encoder) error { return enc.WriteSection(c.name, c.data) },
			func(dec *copyover.Decoder) error { return dec.ReadSection(c.name, &c.restored) },
		))
	}

	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, copyover.Save(w))
	w.Close()

	require.NoError(t, copyover.Restore(int(r.Fd())))
	r.Close()

	for _, c := range contribs {
		assert.Equal(t, c.data, c.restored, "section %q", c.name)
	}
}

// TestDecoder_MissingSection verifies that reading a section that was never
// written returns an error rather than silently producing a zero value.
func TestDecoder_MissingSection(t *testing.T) {
	copyover.ResetRegistry()

	var restored string
	copyover.Register(copyover.FuncContributor(
		"present",
		func(enc *copyover.Encoder) error { return enc.WriteSection("present", "value") },
		func(dec *copyover.Decoder) error {
			// Attempt to read a section that was never saved.
			return dec.ReadSection("absent", &restored)
		},
	))

	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, copyover.Save(w))
	w.Close()

	err = copyover.Restore(int(r.Fd()))
	r.Close()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "absent")
}

// TestSaveRestore_InProcess verifies the full Save→Restore cycle without
// spawning a subprocess. Contributors save known values; after Restore the same
// contributors must hold those values back.
func TestSaveRestore_InProcess(t *testing.T) {
	copyover.ResetRegistry()

	alpha := &captureContributor{name: "alpha", payload: "state-alpha"}
	beta := &captureContributor{name: "beta", payload: "state-beta"}
	copyover.Register(alpha)
	copyover.Register(beta)

	r, w, err := os.Pipe()
	require.NoError(t, err)

	require.NoError(t, copyover.Save(w))
	w.Close()

	require.NoError(t, copyover.Restore(int(r.Fd())))
	r.Close()

	assert.Equal(t, "state-alpha", alpha.restored)
	assert.Equal(t, "state-beta", beta.restored)
}

// TestSaveRestore_Subprocess verifies the end-to-end exec path: Save writes
// into a pipe whose read end is passed to a child process via ExtraFiles
// (becoming fd 3), and the child calls Restore and reports the recovered values
// back over stdout.
func TestSaveRestore_Subprocess(t *testing.T) {
	copyover.ResetRegistry()

	copyover.Register(&captureContributor{name: "alpha", payload: "hello-from-parent"})
	copyover.Register(&captureContributor{name: "beta", payload: "world-from-parent"})

	r, w, err := os.Pipe()
	require.NoError(t, err)

	require.NoError(t, copyover.Save(w))
	w.Close()

	// Re-exec the test binary itself. TestMain will run the helper instead of
	// any test when GO_COPYOVER_HELPER=1.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(),
		"GO_COPYOVER_HELPER=1",
		"GO_COPYOVER_FD=3", // ExtraFiles[0] becomes fd 3 in the child
	)
	cmd.ExtraFiles = []*os.File{r}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	r.Close()
	require.NoError(t, err, "helper stderr: %s", stderr.String())

	var result map[string]string
	require.NoError(t, json.NewDecoder(&stdout).Decode(&result))

	assert.Equal(t, "hello-from-parent", result["alpha"])
	assert.Equal(t, "world-from-parent", result["beta"])
}

// TestSave_ErrorPropagation verifies that a contributor whose CopyoverSave
// returns an error causes Save to fail and surface that error.
func TestSave_ErrorPropagation(t *testing.T) {
	copyover.ResetRegistry()
	copyover.Register(&errorContributor{})

	var buf bytes.Buffer
	err := copyover.Save(&buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "intentional save failure")
}

type errorContributor struct{}

func (e *errorContributor) CopyoverName() string { return "error-contributor" }
func (e *errorContributor) CopyoverSave(_ *copyover.Encoder) error {
	return fmt.Errorf("intentional save failure")
}
func (e *errorContributor) CopyoverRestore(_ *copyover.Decoder) error { return nil }

// TestRestore_TruncatedPipe verifies that Restore returns an error when the
// pipe contains incomplete data rather than silently succeeding or panicking.
func TestRestore_TruncatedPipe(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)

	// Write only 2 bytes of the 4-byte section count header.
	_, err = w.Write([]byte{0x00, 0x01})
	require.NoError(t, err)
	w.Close()

	copyover.ResetRegistry()
	err = copyover.Restore(int(r.Fd()))
	r.Close()

	assert.Error(t, err)
}
