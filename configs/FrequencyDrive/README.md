# Frequency Drive Configs

This folder contains branded frequency-drive mock configurations and vendor audit notes.

Each vendor folder now contains a `README.md` that separates:

- local trusted documents
- official documents identified online
- blocked or missing downloads
- verified behavior
- unverified or simulator-only behavior

## Example Configs

This folder currently contains:

- `delta_ms300_drive_c.json`
- `ABB/abb_fena_drive_c.json`
- `Danfoss/danfoss_fc302_drive_c.json`
- `Delta/delta_ms300_drive_c.json`
- `Invertek/invertek_optidrive_p2_drive_c.json`
- `Omron/omron_mx2_drive_c.json`
- `Schneider/schneider_atv320_drive_c.json`
- `Siemens/siemens_sinamics_drive_c.json`

The repository root example config is:

- `../../delta_ms300_drive_c.json`

## Tracking Table

Use this table as the top-level audit tracker. `Yes` means the item exists and is trusted for that column. `Partial` means the item exists but is incomplete, blocked, or not authoritative enough for full validation. `No` means it is not implemented or not verified yet.

| Vendor | Config Present | Local Official Docs | Official Docs Identified Online | Blocked / Not Downloadable | Main Register Map Backed by Trusted Local PDF/XLSX | Wider Parameter Map | Runtime Behavior in Go | Trusted For Network Validation | Missing Types | Missing Info |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| ABB | Yes | Yes | Partial | No | Partial | No | Yes | Partial | Specific ABB drive-family model | Placeholder registers `16..50`; no verified drive-family parameter map; current config is FENA adapter profile only |
| Danfoss | Yes | Yes | Partial | No | Partial | No | Yes | Partial | Full FC302 parameter model | Only `2810..2819` and `2910..2919` are mapped with confidence; wider parameter space not transcribed |
| Delta | Yes | Yes | Yes | No | Yes | Partial | Yes | Partial | Wider MS300 parameter families | Strongest implementation; command and monitor areas are mapped, but not the full parameter space and no TCP default port/unit-ID evidence was found |
| Invertek | Yes | Yes | Yes | Partial | Partial | No | Yes | Partial | Wider Optidrive parameter model | Only cyclic words `1..4` and `256..259` are mapped with confidence; no vendor-hosted `OPT-2-MODIP-IN` user guide was found |
| Omron | Yes | Yes | Partial | No | Partial | No | No | Partial | Omron MX2 flexible-format profile | The Omron config is based on the option-board flexible-format mapping, but broader MX2 parameter coverage and runtime behavior are still not implemented |
| Schneider | Yes | Yes | Yes | Yes | Partial | Partial | Yes | Partial | Fuller ATV320 object/register model | Workbook-backed core addresses exist and the local folder now contains both `NVE41308` and `NVE41313`, but vendor-hosted scripted downloads were blocked during agent retrieval |
| Siemens | Yes | Yes | Yes | Yes | Partial | No | Yes | Partial | Fuller SINAMICS parameter model | Process data `40100..40119` and selected diagnostics are mapped; broader vendor-hosted support PDFs were identified but blocked |

## Feature Checklist

| Feature | ABB | Danfoss | Delta | Invertek | Omron | Schneider | Siemens |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `*_c.json` exists | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Vendor folder audit `README.md` exists | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Local manuals/workbooks available | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Main cyclic command/status map documented | Partial | Partial | Yes | Partial | Partial | Partial | Partial |
| Additional documented registers included | Partial | Partial | Yes | No | No | Partial | Partial |
| Large placeholder spans still present | Yes | Yes | Yes | No | No | Yes | Yes |
| Basic runtime behavior implemented | Yes | Yes | Yes | Yes | No | Yes | Yes |
| Runtime behavior matches vendor-specific semantics | No | No | Partial | No | No | No | No |
| Simulator port confirmed from docs | No | No | No | No | No | No | No |

## Network Validation

This table separates what is actually backed by the vendor documentation from what is just a simulator choice in the current JSON files.

| Vendor | Protocol In Config | Protocol Backed By Docs | Listen Address In Config | Port In Config | Port Backed By Docs | Unit IDs In Config | Unit IDs Backed By Docs | Serial Framing Fields In Config | Serial Framing Backed By Docs | Network Evidence Outcome | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| ABB | `modbus-tcp` | Yes | `127.0.0.1` | `502` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | ABB docs confirm FENA Modbus TCP support; the config now uses Modbus TCP default port 502 because no ABB-specific port override was verified, while listen address and unit ID remain undocumented defaults |
| Danfoss | `modbus-tcp` | Yes | `127.0.0.1` | `502` | No | `[255]` | Partial | `8 / none / 1` | N/A for Modbus TCP | Partial | The FC302 Modbus TCP application note supports the protocol; the config now uses Modbus TCP default port 502, and the `Unit ID 255` value remains prior-audit evidence rather than a newly re-extracted proof from this pass |
| Delta | `modbus-tcp` | Partial | `127.0.0.1` | `502` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The local Delta docs back the Modbus register map and product communication capability; the config now uses Modbus TCP default port 502 because no Delta-specific port override was verified |
| Invertek | `modbus-tcp` | Yes | `127.0.0.1` | `502` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The local quick-start and user guide identify a Modbus TCP interface; the config now uses Modbus TCP default port 502 because no Invertek-specific port override was verified |
| Omron | `modbus-tcp` | Partial | `127.0.0.1` | `502` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The Omron config follows the documented flexible-format Modbus mapping and now uses Modbus TCP default port 502 because no Omron-specific port override was verified |
| Schneider | `modbus-tcp` | Yes | `127.0.0.1` | `502` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The local folder now contains both Schneider manuals `NVE41308` and `NVE41313`; the config now uses Modbus TCP default port 502 because no Schneider-specific port override was verified |
| Siemens | `modbus-tcp` | Yes | `127.0.0.1` | `502` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The SINAMICS app note supports the Modbus TCP protocol and register areas; the config now uses Modbus TCP default port 502 because no Siemens-specific port override was verified |

## Runtime Framing

- The Go server currently accepts only Modbus TCP MBAP frames with `Protocol Identifier = 0`.
- This is enforced in `internal/modbus/server.go`, not in the vendor JSON files.
- The JSON fields `data_bits`, `parity`, and `stop_bits` are metadata only in the current runtime and should not be read as active TCP transport settings.

## Audit Summary

- `Delta` is the most complete folder and still has the strongest documentation-to-runtime alignment.
- `ABB`, `Danfoss`, `Invertek`, `Schneider`, and `Siemens` all have configs and basic runtime behavior, but they are still partial implementations.
- `Omron` now has a partial MX2 config based on the documented flexible-format mapping, but it still lacks vendor-specific runtime behavior and verified network defaults.
- `Siemens` still has official vendor documents identified online that could not be re-downloaded automatically because the vendor CDN returned access-denied responses from this environment.
- `Schneider` now has the missing local manual set for the current ATV320 audit, even though the Schneider CDN still blocked scripted retrieval from this environment.
- No current simulator `port` value should be treated as vendor-document-verified unless that evidence is added explicitly to the vendor audit notes.
