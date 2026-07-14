package copyover

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Contributor is implemented by any subsystem that needs to save/restore state
// across a copyover.
type Contributor interface {
	// CopyoverName returns a stable, unique key for this contributor.
	CopyoverName() string
	// CopyoverSave serializes state into the provided encoder.
	CopyoverSave(enc *Encoder) error
	// CopyoverRestore deserializes state from the provided decoder.
	CopyoverRestore(dec *Decoder) error
}

// Encoder writes named sections into a stream.
type Encoder struct {
	w io.Writer
}

// WriteSection serializes v as JSON and writes it as a named section.
func (e *Encoder) WriteSection(name string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("copyover: marshal section %q: %w", name, err)
	}

	nameBytes := []byte(name)
	if err := binary.Write(e.w, binary.BigEndian, uint16(len(nameBytes))); err != nil {
		return fmt.Errorf("copyover: write name length for %q: %w", name, err)
	}
	if _, err := e.w.Write(nameBytes); err != nil {
		return fmt.Errorf("copyover: write name for %q: %w", name, err)
	}
	if err := binary.Write(e.w, binary.BigEndian, uint32(len(data))); err != nil {
		return fmt.Errorf("copyover: write data length for %q: %w", name, err)
	}
	if _, err := e.w.Write(data); err != nil {
		return fmt.Errorf("copyover: write data for %q: %w", name, err)
	}
	return nil
}

// Decoder reads named sections from a stream.
type Decoder struct {
	sections map[string][]byte
}

// ReadSection deserializes the named section into v.
func (d *Decoder) ReadSection(name string, v any) error {
	data, ok := d.sections[name]
	if !ok {
		return fmt.Errorf("copyover: section %q not found", name)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("copyover: unmarshal section %q: %w", name, err)
	}
	return nil
}

var registry []Contributor

// Register adds a contributor to the global registry.
// Must be called before copyover is triggered.
func Register(c Contributor) {
	registry = append(registry, c)
}

// ResetRegistry clears all registered contributors. Intended for use in tests.
func ResetRegistry() {
	registry = nil
}

// funcContributor is a Contributor backed by plain functions. Used in tests.
type funcContributor struct {
	name      string
	saveFn    func(*Encoder) error
	restoreFn func(*Decoder) error
}

func (f *funcContributor) CopyoverName() string               { return f.name }
func (f *funcContributor) CopyoverSave(enc *Encoder) error    { return f.saveFn(enc) }
func (f *funcContributor) CopyoverRestore(dec *Decoder) error { return f.restoreFn(dec) }

// FuncContributor returns a Contributor implemented by the provided functions.
// Intended for use in tests.
func FuncContributor(name string, save func(*Encoder) error, restore func(*Decoder) error) Contributor {
	return &funcContributor{name: name, saveFn: save, restoreFn: restore}
}

// Save serializes all registered contributors into w.
func Save(w io.Writer) error {
	enc := &Encoder{w: w}

	if err := binary.Write(w, binary.BigEndian, uint32(len(registry))); err != nil {
		return fmt.Errorf("copyover: write section count: %w", err)
	}

	for _, c := range registry {
		if err := c.CopyoverSave(enc); err != nil {
			return fmt.Errorf("copyover: save %q: %w", c.CopyoverName(), err)
		}
	}

	return nil
}

// Execute serializes all registered contributors into a pipe, launches the
// new process image with the pipe fd passed via --copyover-fd, then exits.
// extraArgs are forwarded as-is to the child process.
// Returns an error on platforms where copyover is not supported.
func Execute(binaryPath string, extraArgs []string) error {
	if runtime.GOOS == "windows" {
		return errors.New("copyover is not supported on this platform")
	}

	readFd, writeFd, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("copyover: os.Pipe: %w", err)
	}

	if err := Save(writeFd); err != nil {
		readFd.Close()
		writeFd.Close()
		return err
	}

	writeFd.Close()

	// ExtraFiles are passed to the child as fd 3, 4, 5, ... in order.
	// readFd becomes fd 3 in the child process.
	args := make([]string, 0, len(extraArgs)+1)
	for _, a := range extraArgs {
		// Strip any existing --copyover-fd flag so it is not duplicated.
		if strings.HasPrefix(a, "--copyover-fd") || strings.HasPrefix(a, "-copyover-fd") {
			continue
		}
		args = append(args, a)
	}
	args = append(args, "--copyover-fd=3")

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{readFd}
	cmd.SysProcAttr = newSysProcAttr()

	if err := cmd.Start(); err != nil {
		readFd.Close()
		return fmt.Errorf("copyover: start child process: %w", err)
	}

	readFd.Close()
	os.Exit(0)
	return nil
}

// Restore reads state from the given file descriptor and calls CopyoverRestore
// on all registered contributors.
func Restore(fd int) error {
	f := os.NewFile(uintptr(fd), "copyover-pipe")
	if f == nil {
		return fmt.Errorf("copyover: invalid fd %d", fd)
	}
	defer f.Close()

	dec, err := readSections(f)
	if err != nil {
		return fmt.Errorf("copyover: read sections: %w", err)
	}

	for _, c := range registry {
		if err := c.CopyoverRestore(dec); err != nil {
			return fmt.Errorf("copyover: restore %q: %w", c.CopyoverName(), err)
		}
	}

	return nil
}

func readSections(r io.Reader) (*Decoder, error) {
	var sectionCount uint32
	if err := binary.Read(r, binary.BigEndian, &sectionCount); err != nil {
		return nil, fmt.Errorf("read section count: %w", err)
	}

	sections := make(map[string][]byte, sectionCount)

	for i := uint32(0); i < sectionCount; i++ {
		var nameLen uint16
		if err := binary.Read(r, binary.BigEndian, &nameLen); err != nil {
			return nil, fmt.Errorf("read name length at section %d: %w", i, err)
		}
		nameBytes := make([]byte, nameLen)
		if _, err := io.ReadFull(r, nameBytes); err != nil {
			return nil, fmt.Errorf("read name at section %d: %w", i, err)
		}

		var dataLen uint32
		if err := binary.Read(r, binary.BigEndian, &dataLen); err != nil {
			return nil, fmt.Errorf("read data length at section %d: %w", i, err)
		}
		data := make([]byte, dataLen)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("read data at section %d: %w", i, err)
		}

		sections[string(nameBytes)] = data
	}

	return &Decoder{sections: sections}, nil
}
