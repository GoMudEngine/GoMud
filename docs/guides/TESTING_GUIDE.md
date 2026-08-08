# Testing DOGMud

## Focused development tests

Use native `go test` commands for fast feedback on a package or named test.
Native Windows execution may still be affected by antivirus handling of
generated Go test binaries.

## Reproducible Linux race baseline

Prerequisites:

- Docker Desktop or another running Linux-container Docker engine;
- Docker Compose v2;
- a host architecture supported by Go's Linux race detector;
- sufficient disk and memory for a race build; and
- registry and Go-module network access for the first uncached image build.

Downloaded module dependencies are baked into the resulting test image.
Individual application tests may still use networking when their behavior
requires it.

From the repository root, run:

```text
docker compose -f compose.test.yml run --build --rm test
```

The build copies the filtered current working tree into an image. Compose then
starts a fresh container and runs:

```text
go test -v -count=1 -timeout 300s -race ./...
```

Authoritative success evidence is both process exit zero and the complete
`go test -v` terminal output, including visible skips. Exit code alone is not
sufficient. `-count=1` prevents successful Go test results from being reused.
The timeout applies to each package test binary, not to the aggregate Docker
build and run.

Tests requiring `DOGMUD_BOOT_SMOKE` remain visible as skips. Selected smoke
tests run separately in pull-request CI; assigning the remaining opt-in gates
belongs to Roadmap Chunk 1.1.

Ubuntu pull-request CI runs:

```text
go test -timeout 300s -race -coverprofile=coverage.out ./...
```

The local baseline uses the same package pattern, per-package timeout, and race
detector. It adds `-v` and `-count=1` and omits the coverage profile. Broader
CI parity, including formatting, vet, lint, and other workflow gates, belongs
elsewhere; this command is not complete CI parity.

## Failure categories

- Docker daemon unavailable: start the Linux-container engine.
- Image pull or build failure: inspect the failing Dockerfile step.
- Generation failure: fix the reported `go generate ./...` error.
- Compilation or test failure: use the named package and test output.
- Timeout: identify the package whose test binary exceeded 300 seconds.
- Race report: treat it as a failing test and preserve the detector output.

Do not disable antivirus, sandboxing, or other security controls to force a
native Windows run to pass.

## Reproducibility boundary

The Go version is aligned to `go.mod`, but the Docker base image is selected by
a mutable version tag rather than a digest. This is a repeatable,
version-aligned Linux environment, not a bit-for-bit hermetic build.

## Ephemeral playtest supervisor (`playtestenv`)

Chunk 0.3a adds a local-only Go supervisor (`cmd/playtestenv` over
`internal/playtestenv`) that builds one selected checkout into one disposable,
lease-bound Docker server on a Docker-assigned loopback AI port. It cannot
target production or any remote host. Runtime admin/builder mutations live in a
disposable volume and are discarded with the run; the checkout is never mounted
or exported back into source.

### Package unit tests

From the repository root:

```powershell
go test ./cmd/playtestenv ./internal/playtestenv ./internal/playtestprofiles
```

These exercise fake runners, lifecycle contracts, and profile materialization
without starting Docker.

### Synthetic profiles (chunk 0.3b)

Tracked templates live in `tools/playtest/profiles/` (`fresh`, `early`, `mid`,
`veteran`, `specialist-caster`, `admin`). The runner image copies them to
`/app/playtest/profiles`.

Request profiles on start (library API or CLI). Example CLI:

```powershell
go run ./cmd/playtestenv start --checkout . --profile fresh:5200 --json
```

When profiles are requested, the supervisor writes
`tools/playtest/.run/<run-id>/control/profiles-manifest.yaml` and Playtest
config overrides. After `ready`, `artifacts.creds` points at
`.../control/creds.json` (mode 0600 where the OS allows). That file holds
per-run usernames/passwords for AI-port login — **do not commit it** (`.run/`
is gitignored) and never paste password bodies into failure reports.

Empty/omitted `--profile` flags leave `ProfilesManifest` unset (creation-flow
smoke: agent creates a character). Bad `start_room` / overlay refs fail the
container before `Server Ready`.

### Chunk 0.2 full-suite baseline (still required)

Supervisor work does not replace the Linux-container race baseline:

```powershell
docker compose -f compose.test.yml run --build --rm test
```

Do not substitute an unreliable native-Windows `go test ./...` for this command.

### Opt-in real-Docker integration

Integration tests are gated and slow. They require Docker Desktop (or another
Linux-container engine), a **named local Docker context**, Compose **>= 2.20.0**,
and an unset `DOCKER_HOST`. On Windows the context endpoint must be `npipe://`;
on Linux it must be `unix://`.

```powershell
$env:DOGMUD_PLAYTESTENV_INTEGRATION = "1"
go test -v -run "^TestDockerIntegration$" -timeout 30m ./internal/playtestenv
Remove-Item Env:\DOGMUD_PLAYTESTENV_INTEGRATION
```

Without the env var, the integration suite skips. Tests may delete only
resources that carry the exact test-created identity and complete supervisor
labels.

### VERSION pin and writable control overrides

The selected checkout's `main.go` must declare a package-level string
`VERSION` constant. The supervisor pins that value into disposable control
overrides under `tools/playtest/.run/<run-id>/control/` so the ephemeral image
boots current-data for that version. The control directory must be writable;
failures surface as structured checkout/control errors rather than silent
misboots.

### Lease, renew, and reap

- `start` grants a lease (default two hours).
- `renew` extends a ready, unexpired run whose live Docker labels still agree
  with the manifest.
- `reap` walks `tools/playtest/.run/` under the checkout, cleans only
  lease-expired runs with unambiguous label/manifest agreement, and reports
  other labelled leftovers as diagnostics **without** deleting them.

Advisory lock contention is retryable (`lock_busy`). Ambiguous identity never
widens deletion.

### Inspecting leftovers

After integration or a failed cleanup, inspect managed residue with the active
local context (example uses the current context name):

```powershell
if ($env:DOCKER_HOST) { throw "unset DOCKER_HOST and select a named local context" }
$localContext = (docker context show).Trim()
docker --context $localContext ps -a --filter "label=dogmud.playtest.managed=true"
docker --context $localContext network ls --filter "label=dogmud.playtest.managed=true"
docker --context $localContext volume ls --filter "label=dogmud.playtest.managed=true"
docker --context $localContext image ls --filter "label=dogmud.playtest.managed=true"
```

Expected after a clean run: no test-owned resources. Labelled decoys retained
intentionally by a test must be removed by that test's exact cleanup, not a
broad wipe. Host shop/guild/moderation persistence is never touched; the image
uses its own data volume.

Run artifacts under `tools/playtest/.run/` and failure reports under
`tools/playtest/reports/` must remain Git-ignored.
