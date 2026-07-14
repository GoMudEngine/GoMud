# Copyover

Copyover is a live-restart mechanism: the server re-executes its own binary
without dropping active TCP connections. Players experience a brief pause
followed by a "Copyover complete." message rather than a full disconnect and
reconnect cycle.

Copyover is supported on Unix-like platforms only. The feature compiles to a
no-op on Windows.

## How it works

### Trigger

A copyover can be initiated in several ways:

1. **In-game command** - an admin runs the `copyover` command
   (`internal/usercommands/admin.copyover.go`).
2. **Signal** - sending `SIGUSR1` to the running process
   (`copyover_signal_unix.go`).
   ```
   $ kill -SIGUSR1 <pid>
   ```

All methods ultimately call `triggerCopyover()` in `copyover.go` (package
`main`), which:

1. Sets `serverAlive` to `false` to stop accepting new connections.
2. Flushes rooms, users, and plugins to disk.
3. Calls `copyover.Execute(binaryPath, os.Args[1:])`.

### State serialization (`copyover.Execute`)

`Execute` (in `internal/copyover/copyover.go`) performs the following steps:

1. Opens an `os.Pipe()` — a `(readFd, writeFd)` pair.
2. Calls `copyover.Save(writeFd)`, which iterates every registered
   `Contributor` and calls `CopyoverSave` on each one. Each contributor writes
   a named section into the pipe using `Encoder.WriteSection`.
3. Closes `writeFd` (signals EOF to the reader).
4. Launches the same binary again via `exec.Command`, passing the read end of
   the pipe as `ExtraFiles[0]` — the OS assigns it file descriptor 3 in the
   child — and appends `--copyover-fd=3` to the argument list.
5. Calls `os.Exit(0)`, terminating the parent.

#### Wire format

The binary stream written by `Save` is:

```
[uint32 big-endian] section count
for each section:
  [uint16 big-endian] length of name (bytes)
  [N bytes]           name (UTF-8)
  [uint32 big-endian] length of JSON payload (bytes)
  [M bytes]           JSON payload
```

All payloads are JSON-marshalled Go structs.

### Connection hand-off

Before serializing, the `connections` contributor (`internal/connections/`)
clears the `FD_CLOEXEC` flag on each raw TCP socket so the file descriptor
survives the `exec`. WebSocket connections cannot be transferred and receive a
disconnect notice instead.

### State restoration (`copyover.Restore`)

When the child process starts, `main()` checks `flags.CopyoverFd()`. If the
flag is >= 0, this is a copyover restart. After registering all contributors,
`main` calls `copyover.Restore(fd)`, which:

1. Reads all sections from the pipe into an in-memory map (keyed by name).
2. Iterates every registered `Contributor` and calls `CopyoverRestore` on
   each, passing a `Decoder` that looks up sections by name.

After `Restore` returns, `main` loops over all restored connections and spawns
a `resumeRestoredConnection` goroutine for each one, picking up the I/O loop
exactly where the old process left it.

Normal startup steps that would corrupt restored state (user index rebuild,
round-count load from disk, data migrations) are skipped when
`flags.CopyoverFd() >= 0`.

## The `Contributor` interface

```go
type Contributor interface {
    CopyoverName() string
    CopyoverSave(enc *Encoder) error
    CopyoverRestore(dec *Decoder) error
}
```

| Method | Responsibility |
|---|---|
| `CopyoverName` | Returns a stable, unique string key for this contributor. Changing this key is a breaking change. |
| `CopyoverSave` | Serializes in-memory state into the encoder. Called once per copyover in the parent process. |
| `CopyoverRestore` | Deserializes state from the decoder. Called once per copyover in the child process. |

`Encoder.WriteSection(name, v)` marshals `v` as JSON and appends it to the
stream. `Decoder.ReadSection(name, &v)` looks up the section by name and
unmarshals it into `v`. Reading a section that was never written is an error.

## Implementing a new contributor

### 1. Define a state struct

Create a struct that holds only the in-memory values that cannot be derived
from disk after a restart. Use JSON tags.

```go
type myState struct {
    Counter uint64 `json:"counter"`
    Label   string `json:"label"`
}
```

### 2. Implement the contributor

```go
type myContributor struct{}

func (m *myContributor) CopyoverName() string { return "mypackage" }

func (m *myContributor) CopyoverSave(enc *copyover.Encoder) error {
    return enc.WriteSection(m.CopyoverName(), myState{
        Counter: currentCounter,
        Label:   currentLabel,
    })
}

func (m *myContributor) CopyoverRestore(dec *copyover.Decoder) error {
    var state myState
    if err := dec.ReadSection(m.CopyoverName(), &state); err != nil {
        return err
    }
    currentCounter = state.Counter
    currentLabel   = state.Label
    return nil
}

func CopyoverContributor() copyover.Contributor {
    return &myContributor{}
}
```

Convention: place this in a file named `copyover.go` inside your package, and
expose a `CopyoverContributor()` factory function.

### 3. Register the contributor in `main.go`

Contributors must be registered before `copyover.Restore` is called. Add a
call alongside the existing registrations near the top of `main()`:

```go
copyover.Register(mypackage.CopyoverContributor())
```

The order of registration does not affect correctness (sections are looked up
by name), but keeping the list in a consistent order makes the code easier to
read.

### 4. Handle platform differences if needed

If your contributor relies on OS-specific mechanisms (e.g., raw file
descriptors), provide separate `copyover.go` build-tagged files:

- `//go:build !windows` — full implementation
- `//go:build windows` — stub that writes/reads an empty or minimal state

See `internal/connections/fd_unix.go` and `internal/connections/fd_windows.go`
for a concrete example.

## Plugins

Plugins must not implement the `Contributor` interface or call
`copyover.Register`. The copyover mechanism is an internal server facility
reserved for core subsystems.

Plugins have their own persistence callbacks that are already invoked at the
right points during a copyover:

- `PluginCallbacks.SetOnSave(f func())` - called by `plugins.Save()` before
  the new process is launched.
- `PluginCallbacks.SetOnLoad(f func())` - called by `plugins.Load()` when the
  new process initialises.

Plugin state written during `onSave` (via `Plugin.WriteBytes`,
`Plugin.WriteStruct`, etc.) is persisted to disk under
`<datafiles>/plugin-data/` and is available to read in `onLoad` after the
restart. This is the correct pattern for plugin state that must survive a
copyover.

## Guidelines

- **Keep state minimal.** Only serialize values that are expensive or
  impossible to reconstruct from disk. Anything that can be reloaded from data
  files should be left to normal startup.
- **Use stable names.** `CopyoverName()` is the key used to locate a section
  in the stream. Renaming it breaks in-flight copyovers.
- **Treat restore errors as fatal.** If `CopyoverRestore` returns an error,
  `main` calls `os.Exit(1)`. Do not swallow errors silently; surface them so
  the operator knows the restart failed.
- **No side effects in `CopyoverSave`.** The parent process is about to exit.
  Do not start goroutines, open files, or modify shared state inside
  `CopyoverSave`.
- **Test both paths.** Use `copyover.FuncContributor` and the
  `Save`/`Restore` helpers in tests to verify your contributor round-trips
  correctly without spawning a subprocess. See `internal/copyover/copyover_test.go`
  for examples.
