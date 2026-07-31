# UUID Context

## Purpose

`internal/uuid` is a bespoke 128-bit identifier. It is **not** RFC 4122 — the
layout packs a microsecond timestamp, a sequence counter, a version nibble, and
an 8-bit type tag, so an id sorts chronologically by raw bytes and reports what
kind of thing it names without a lookup.

## Files

- **uuid.go** — the `UUID` type, bit-field accessors, and the generator.
- **uuid_string.go** — versioned string encode/decode (`FromString`, plus the
  v1 codec).

## Bit layout

Big-endian, bits 127→0:

| Bits    | Width | Field                |
|---------|-------|----------------------|
| 127–76  | 52    | Timestamp (µs since custom epoch) |
| 75–68   | 8     | Sequence             |
| 67–64   | 4     | Version              |
| 63–60   | 4     | Type, high nibble    |
| 59–56   | 4     | Type, low nibble     |
| 55–0    | 56    | Unused               |

**The type field is split across the two 8-byte halves.** `Type()` reads the
high nibble from the top word's low bits and the low nibble from the bottom
word's top bits, then recombines. Anything that rewrites the encoding must
preserve that split or every existing id changes meaning.

`customEpoch` is 2025-01-01 (`1735689600000000` µs). The 52-bit timestamp gives
roughly 142 years of range from there; the 8-bit sequence allows 256 ids per
microsecond.

## Public API

```go
type IDType uint8
type UUID [16]byte

func New(typeID ...IDType) UUID          // package-level generation entry point
func FromString(s string) (UUID, error)

func (u UUID) Time() time.Time
func (u UUID) Timestamp() uint64
func (u UUID) Sequence() uint8
func (u UUID) Version() uint8
func (u UUID) Type() IDType
func (u UUID) Unused() uint64
func (u UUID) IsNil() bool
func (u UUID) String() string
func (u UUID) MarshalText() ([]byte, error)
func (u *UUID) UnmarshalText(text []byte) error
```

`MarshalText`/`UnmarshalText` are what make a `UUID` round-trip through YAML
and JSON as a plain string rather than a 16-element byte array.

## Generator

```go
type UUIDGenerator struct{ /* mutex + last timestamp + sequence */ }
func (g *UUIDGenerator) NewUUID(typeID IDType) UUID
```

A single package-level `generator` singleton, mutex-guarded, created at init.
`New` delegates to it — call `New`, not the method. `currentVersion` is 1 and
selects the string codec (`toString_v1` / `fromString_v1`), so bumping the
version means adding a codec pair, not editing the existing one.

## Gotchas

- **There is no `ParseUUID` or `MustParseUUID`.** Parsing is `FromString`, and
  it returns an error you must handle — there is no panic-on-error variant.
- **`New` takes a variadic type id.** Calling `New()` with no argument is legal
  and yields type 0; be explicit.
- **Sorting by bytes sorts by time, but only within one version.** The version
  nibble sits below the timestamp, so mixed-version ids still order correctly by
  time — do not rely on that if a future version moves the field.
- **The timestamp is visible to anyone holding the id.** Generation time and
  entity type are recoverable from the id itself; do not use one where that
  leak matters.

## Dependencies

Standard library only: `encoding/binary`, `strconv`, `sync`, `time`.
