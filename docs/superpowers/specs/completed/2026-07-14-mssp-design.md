# MSSP (MUD Server Status Protocol) — Design

**Date:** 2026-07-14
**Status:** Approved (brainstorm), ready for implementation plan
**Goal:** Advertise DOGMud correctly on MUD listing sites (TMC, MudConnect, Grapevine,
mudstats, etc.) by responding to the MSSP telnet handshake with a rich set of
server-status fields. Without MSSP, DOGMud lists as a blank/dead entry nobody clicks —
this is the single highest-ROI item on the path to 1.0.

---

## 1. Background

MSSP is a telnet sub-negotiation option (**option byte 70**) that crawlers use to read
a server's live stats and descriptive metadata. Flow:

1. Client (crawler) connects.
2. Server offers the option: `IAC WILL MSSP`.
3. Crawler accepts: `IAC DO MSSP`.
4. Server replies once with the data block:
   `IAC SB MSSP  (MSSP_VAR name MSSP_VAL value)…  IAC SE`
   where `MSSP_VAR = 1` and `MSSP_VAL = 2`.

The reply is a snapshot; crawlers reconnect periodically for fresh player counts.

DOGMud already implements sibling telnet options the same way — **MSP** (byte 90, sound)
and GMCP — so all the plumbing exists. MSSP mirrors the MSP pattern exactly.

---

## 2. Architecture

Purely additive. Two new small files + two one-spot edits. No existing behavior changes.

### 2.1 `internal/term/mssp.go` (new) — the byte protocol (pure, no deps)
Mirrors `internal/term/msp.go`. Defines:
- `MSSP IACByte = 70`
- `MSSP_VAR byte = 1`, `MSSP_VAL byte = 2`
- `MsspEnable  = TerminalCommand{[]byte{TELNET_IAC, TELNET_WILL, MSSP}, []byte{}}`
- `MsspAccept  = TerminalCommand{[]byte{TELNET_IAC, TELNET_DO,   MSSP}, []byte{}}` (client accepts)
- `MsspRefuse  = TerminalCommand{[]byte{TELNET_IAC, TELNET_DONT, MSSP}, []byte{}}`
- `MsspCommand = TerminalCommand{[]byte{TELNET_IAC, TELNET_SB,   MSSP}, []byte{TELNET_IAC, TELNET_SE}}`
- `func IsMSSPCommand(b []byte) bool` — `len(b) > 2 && b[0]==TELNET_IAC && b[2]==MSSP`
- `type MSSPField struct { Name string; Values []string }`
- `func EncodeMSSPPayload(fields []MSSPField) []byte` — emits, for each field:
  `MSSP_VAR + name-bytes + (MSSP_VAL + value-bytes)…`. Multiple values per field are
  supported (one `MSSP_VAL` per value — MSSP's native multi-value form, used for
  `GAMEPLAY`). **IAC-escaping:** any `0xFF` (IAC) byte inside a name or value is doubled
  (`IAC IAC`) so it can't corrupt the sub-negotiation stream. (Field names/values are
  ASCII in practice, but escaping is correct and cheap.)

The final wire bytes are `MsspCommand.BytesWithPayload(EncodeMSSPPayload(fields))`.

### 2.2 `internal/inputhandlers/mssp.go` (new) — data assembly (testable)
- `func buildMSSPFields(in MSSPInputs) []term.MSSPField` — **pure**. Takes an inputs
  struct (config values, online count, server-start unix time, world counts) and returns
  the ordered field list. This is the unit-tested core.
- `func gatherMSSPFields() []term.MSSPField` — thin wrapper that reads the live inputs
  (`configs.GetServerConfig().MSSP`, `len(users.GetOnlineUserIds())`,
  `util.GetServerStartUnix()`, registry counts) and calls `buildMSSPFields`.

### 2.3 `internal/inputhandlers/term_iac.go` (edit) — respond to the handshake
Add an MSSP block mirroring the existing MSP block (after it): when `IsMSSPCommand(iacCmd)`
and `term.Matches(iacCmd, term.MsspAccept)` (the crawler's `IAC DO MSSP`), send
`term.MsspCommand.BytesWithPayload(term.EncodeMSSPPayload(gatherMSSPFields()))` to the
connection. A `DONT`/refuse is a silent no-op (nothing to disable — we only ever reply on
request).

### 2.4 `main.go` (edit) — offer the option + expose uptime
- In the connect-negotiation batch (~line 617, right after `MspEnable`), send
  `term.MsspEnable.BytesWithPayload(nil)` (`IAC WILL MSSP`). Same connection scope as the
  existing MSP/charset offers.
- After `serverStartTime := time.Now()` (~line 121), call `util.SetServerStart(serverStartTime)`.

### 2.5 `internal/util` (edit) — uptime accessor
- `func SetServerStart(t time.Time)` / `func GetServerStartUnix() int64` — a package var
  holding the server start time so the MSSP assembler can report `UPTIME`. (MSSP `UPTIME`
  is the **unix timestamp the server started**, not a duration.)

### Data flow
```
crawler ──IAC DO MSSP──▶ term_iac.go
                            │ gatherMSSPFields()
                            │   ├─ configs.GetServerConfig().MSSP   (static)
                            │   ├─ len(users.GetOnlineUserIds())    (PLAYERS)
                            │   ├─ util.GetServerStartUnix()        (UPTIME)
                            │   └─ registry counts                  (ROOMS/MOBILES/OBJECTS/…)
                            │ buildMSSPFields() -> []MSSPField
                            │ term.EncodeMSSPPayload(...)
                            ▼
crawler ◀─IAC SB MSSP …data… IAC SE─── connection
```

---

## 3. Fields

Rich set (a fuller listing = more clicks from ads). Three sources:

### Live (computed per request)
| MSSP field | Source |
|---|---|
| `PLAYERS` | `len(users.GetOnlineUserIds())` |
| `UPTIME`  | `util.GetServerStartUnix()` |

### Auto-derived (engine already knows)
| Field | Value / source |
|---|---|
| `CODEBASE` | `GoMud` (constant) |
| `ROOMS` | loaded room count |
| `MOBILES` | loaded mob-template count |
| `OBJECTS` | loaded item-spec count |
| `SKILLS` | `10` (DOGMud skill count) |
| `RACES` | loaded species count |
| `ANSI` | `1` |
| `GMCP` | `1` |
| `MSP` | `1` |
| `UTF-8` | `1` |
| `VT100` | `1` |
| `XTERM 256 COLORS` | `1` |
| `MCCP` | `0` |
| `SSL` | `0` (telnet is unencrypted) |
| `PAY TO PLAY` | `0` |
| `PAY FOR PERKS` | `0` |

> World counts wire to the loaded-registry sizes reported at boot (`rooms.LoadDataFiles`
> etc.). The plan identifies the exact count accessor per package; if a package lacks a
> cheap accessor, the plan adds a one-line `Count()`. Any count that resolves to 0 is
> omitted rather than advertised as empty.

### Config `Server.MSSP` (static — new config block)
| Field | Committed default | Notes |
|---|---|---|
| `Enabled` | `true` | Master toggle for the whole option. |
| `NAME` | (from existing `MudName`) | Not duplicated in the block; assembler reads `MudName`. |
| `WEBSITE` | `https://www.dogmud.org` | |
| `GENRE` | `Fantasy` | |
| `GAMEPLAY` | `Adventure`, `Roleplaying` | Multi-value field. |
| `STATUS` | `Open Beta` | Pre-1.0 positioning. |
| `LANGUAGE` | `English` | |
| `FAMILY` | `Custom` | GoMud is not a DIKU/LP family. |
| `LOCATION` | `United States` | Droplet region. |
| `CREATED` | `2026` | |
| `CONTACT` | **`` (empty)** | **Privacy: NOT defaulted to a personal email in the public repo.** Omitted from the reply when empty. User sets it (droplet `config-overrides.yaml` or `config.yaml`) if they want a public contact. |
| `HOSTNAME` | `` (empty) | Optional canonical connect host; omitted when empty (crawlers use the connecting socket). |
| `PORT` | `` (empty) | Optional canonical telnet port; omitted when empty. |

> **Deliberate deviation from the brainstorm:** `CONTACT` was proposed as
> `calkdavis@gmail.com` and the user said "rest looks good," but committing a personal
> email to a **public** GitHub repo (`pruuk/DOGMud`) is a privacy leak. So the committed
> default is empty and the field is simply omitted from the MSSP reply until the user opts
> in via config. Same treatment for `HOSTNAME`/`PORT` (unknown public values).

Empty config values are **omitted** from the reply (no empty `MSSP_VAL`).

---

## 4. Config schema

New sub-struct on `ServerConfig` (`internal/configs/config.server.go`):

```go
type MSSPConfig struct {
    Enabled  ConfigBool     `yaml:"Enabled"`
    Website  ConfigString   `yaml:"Website"`
    Genre    ConfigString   `yaml:"Genre"`
    Gameplay []ConfigString `yaml:"Gameplay"`
    Status   ConfigString   `yaml:"Status"`
    Language ConfigString   `yaml:"Language"`
    Family   ConfigString   `yaml:"Family"`
    Location ConfigString   `yaml:"Location"`
    Created  ConfigString   `yaml:"Created"`
    Contact  ConfigString   `yaml:"Contact"`
    Hostname ConfigString   `yaml:"Hostname"`
    Port     ConfigString   `yaml:"Port"`
}
```
Field type conventions (`ConfigString`/`ConfigBool`) and the defaulting hook follow the
existing config package pattern (the plan reads a neighboring sub-config for the exact
idiom). Defaults set in the server-config default routine. The block is added to
`_datafiles/config.yaml` under `Server:` with the committed defaults above.

---

## 5. Edge cases & error handling

- **Disabled:** when `Server.MSSP.Enabled` is false, do **not** send `WILL MSSP` on connect
  and ignore any inbound `DO MSSP`. Whole feature is inert.
- **Crawler never sends DO:** we sent `WILL`; if no `DO` arrives, we simply never reply.
  No state kept, no leak.
- **Refuse (`DONT MSSP`):** silent no-op.
- **IAC in a value:** escaped (`IAC IAC`) by the encoder. Prevents stream corruption.
- **Web/websocket connections:** the `WILL MSSP` rides the same negotiation path as the
  existing `MspEnable`; non-telnet clients ignore it exactly as they ignore MSP today.
- **Snapshot semantics:** `PLAYERS`/`UPTIME` reflect the moment of reply. Correct for MSSP.

---

## 6. Testing

**Unit (no network):**
- `TestEncodeMSSPPayload` — one field, multiple fields, a multi-value field (`GAMEPLAY`),
  and IAC-escaping of a `0xFF` in a value. Asserts exact `MSSP_VAR/MSSP_VAL` byte layout.
- `TestBuildMSSPFields` — given a fake `MSSPInputs` (config values, online count 3, a fixed
  start time, fixed world counts): asserts the field list contains `PLAYERS=3`, `UPTIME=<ts>`,
  the configured static values, and that **empty config fields (Contact/Hostname/Port) are
  omitted**, and disabled-genre etc. behave.
- `TestMSSP_DisabledOmitsEverything` (assembler-level) — `Enabled=false` path.

**Manual (on prod after deploy):**
- Connect with a crawler-style probe (a tiny script that sends `IAC DO MSSP` and prints the
  reply) and confirm the field block.
- Validate against a public MSSP checker / confirm the listing renders on one directory.

---

## 7. Out of scope / future

- MSDP / MXP / MCCP (compression) — separate options, not needed for listings now.
- `ICON` (logo URL) — could point at a dogmud.org asset later.
- Auto-detecting `SSL`/`MCCP` from actual capability — hardcoded `0` for now (accurate).
- Registering with each directory site — a manual, non-code follow-up once MSSP is live.

---

## 8. Success criteria

- A crawler sending `IAC DO MSSP` receives a well-formed `IAC SB MSSP …IAC SE` block.
- The block reports live `PLAYERS`/`UPTIME` and the rich static/derived field set.
- No personal contact info is committed to the public repo (opt-in via config).
- Existing connections/clients are unaffected (purely additive negotiation).
- DOGMud renders as a real, populated entry on at least one MUD listing site.
