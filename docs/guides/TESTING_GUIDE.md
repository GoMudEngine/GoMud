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
