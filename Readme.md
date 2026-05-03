# Modbus TCP Simulator

If you work with Modbus TCP, this project is built to become one of your most useful development tools.

`Modbus TCP Simulator` is a practical simulator for testing PLC software against virtual devices, especially frequency drives. It was created to solve a very real engineering problem: developing PLC software without always having the physical Modbus TCP device on the desk.

I built this project while working on industrial control software and needed a reliable way to simulate Modbus TCP based drives during development. The result is a configurable digital twin style simulator that can stand in for real devices and make software development, debugging, and integration much faster.

## Why This Exists

When you are building software for PLCs, hardware is not always available when you need it.

That becomes painful very quickly when:

- The physical device gives back no information whatsoever about what is going wrong!
- You need a debug tool that gives back a lot of info!
- the real frequency drive is not installed yet
- the machine is not assembled
- the device is in use elsewhere
- you want repeatable test conditions
- you need to debug communication without touching production hardware


This simulator helps close that gap.

## What It Does

- simulates Modbus TCP devices using JSON configuration files (_c.json pattern)
- scans the application directory and subdirectories for device configs
- presents a menu so you can choose which server profile to start
- supports multiple branded drive profiles
- provides live logging
- lets you export logs
- shows a register map during runtime
- can be launched multiple times to simulate multiple devices

## Supported Device Profiles

The project currently includes example configurations for:

- ABB
- Danfoss
- Delta
- Invertek
- Schneider
- Siemens
- Omron

These are intended for simulation, testing, and development workflows. I do not claim ownership of any vendor intellectual property, manuals, or register definitions. Vendor names are used only to identify compatible mock profiles.

## Configuration

The simulator uses JSON configuration files to define device behavior and register maps.

The application searches for config files in its own directory tree, making it easy to keep multiple device definitions together in one place. This makes it straightforward to build a test bench with several simulated drives.

## Typical Use Cases

- PLC software development
- Modbus TCP integration testing
- simulating frequency drives before hardware is available
- debugging reads, writes, status words, and register behavior
- creating repeatable test environments
- testing customer-specific device mappings

## Running The Simulator

Start the application and choose the device you want to simulate from the startup menu.

During runtime, you can:

1. View the log
2. Export the log
3. Inspect the register map
4. Exit the simulator

Example runtime menu:
Step 1:
```text
Choose a server configuration:
  1. ABB FENA Ethernet Adapter Drive Mock | frequency-drive | 127.0.0.1:1510
  2. Danfoss VLT FC302 Drive Mock | frequency-drive | 127.0.0.1:1511
  3. Delta MS300 Drive Mock | frequency-drive | 127.0.0.1:1505
  4. Delta MS300 Drive Documentation Example | frequency-drive | 127.0.0.1:1503
  5. Invertek Optidrive P2 Drive Mock | frequency-drive | 127.0.0.1:1512
  6. Schneider Altivar ATV320 Drive Mock | frequency-drive | 127.0.0.1:1513
  7. Siemens SINAMICS Drive Mock | frequency-drive | 127.0.0.1:1514
Enter selection [1-7] or q to quit: 3
```

Step 2:
```text
Server: Delta MS300 Drive Mock | frequency-drive | 127.0.0.1:1505

1. Log
2. Export log
3. Register Map
4. Exit
```
Example's of errors that take way more time to debug!
One of the most time consuming errors is connected to unit id's.
The device gives errors, but the connection is setup correctly. 


```text
2026/05/03 13:12:10.477335 rejected unit id 0 from 127.0.0.1:35233
or
2026/05/03 11:06:05.041363 request tx=0 unit=1 func=0x03 payload=20 00 00 0A
2026/05/03 11:06:05.041363 read registers error: address range 8192..8201 outside block
```

This is now very easy to debug




## Project Philosophy

This is an engineering tool first.

It is meant to be useful, practical, configurable, and easy to extend. It does not try to be a perfect emulation of every real device. Instead, it aims to provide enough realistic behavior to let you develop and verify PLC and control software efficiently.

## Want Another Device Added?

If you want support for another drive or device, open an issue or ticket.

Please include the relevant Modbus documentation. The better the documentation, the better the simulator profile can be.

## Author

Created by **Roelof Jan Boer**.

I invent and build machines, develop industrial software, and create tools that make automation work more practical. This simulator has already been invaluable in my own development workflow, and I hope it helps others just as much.




## Tracking Table

Use this table as the top-level audit tracker. `Yes` means the item exists and is trusted for that column. `Partial` means the item exists but is incomplete, blocked, or not authoritative enough for full validation. `No` means it is not implemented or not verified yet.

| Vendor | Config Present | Local Official Docs | Official Docs Identified Online | Blocked / Not Downloadable | Main Register Map Backed by Trusted Local PDF/XLSX | Wider Parameter Map | Runtime Behavior in Go | Trusted For Network Validation | Missing Types | Missing Info |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| ABB | Yes | Yes | Partial | No | Partial | No | Yes | Partial | Specific ABB drive-family model | Placeholder registers `16..50`; no verified drive-family parameter map; current config is FENA adapter profile only |
| Danfoss | Yes | Yes | Partial | No | Partial | No | Yes | Partial | Full FC302 parameter model | Only `2810..2819` and `2910..2919` are mapped with confidence; wider parameter space not transcribed |
| Delta | Yes | Yes | Yes | No | Yes | Partial | Yes | Partial | Wider MS300 parameter families | Strongest implementation; command and monitor areas are mapped, but not the full parameter space and no TCP default port/unit-ID evidence was found |
| Invertek | Yes | Yes | Yes | Partial | Partial | No | Yes | Partial | Wider Optidrive parameter model | Only cyclic words `1..4` and `256..259` are mapped with confidence; no vendor-hosted `OPT-2-MODIP-IN` user guide was found |
| Omron | Yes | Yes | Partial | No | Partial | No | No | Partial | Omron MX2 flexible-format profile | The Omron config is based on the option-board flexible-format mapping, but broader MX2 parameter coverage and runtime behavior are still not implemented |
| Schneider | Yes | Partial | Yes | Yes | Partial | Partial | Yes | Partial | Fuller ATV320 object/register model | Workbook-backed core addresses exist; `NVE41308` and `NVE41313` are identified on official Schneider pages, but vendor-hosted scripted downloads were blocked |
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
| ABB | `modbus-tcp` | Yes | `127.0.0.1` | `1510` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Simulator default | ABB docs confirm FENA Modbus TCP support, but the localhost bind, chosen port, and unit ID are not tied to a documented ABB setting |
| Danfoss | `modbus-tcp` | Yes | `127.0.0.1` | `1511` | No | `[255]` | Partial | `8 / none / 1` | N/A for Modbus TCP | Partial | The FC302 Modbus TCP application note supports the protocol; the simulator port is local, and the `Unit ID 255` value remains prior-audit evidence rather than a newly re-extracted proof from this pass |
| Delta | `modbus-tcp` | Partial | `127.0.0.1` | `1505` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The local Delta docs back the Modbus register map and product communication capability, but no clean Modbus TCP default port or unit-ID statement was found |
| Invertek | `modbus-tcp` | Yes | `127.0.0.1` | `1512` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The local quick-start and user guide identify a Modbus TCP interface, but the chosen port and unit ID are not verified from the documents |
| Omron | `modbus-tcp` | Partial | `127.0.0.1` | `1515` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The Omron config follows the documented flexible-format Modbus mapping, but the network defaults remain simulator-only and no Omron-specific runtime behavior is implemented |
| Schneider | `modbus-tcp` | Yes | `127.0.0.1` | `1513` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | Official Schneider pages identify `NVE41308` and `NVE41313`, but vendor-hosted scripted downloads were blocked, so the config still uses simulator-local addressing and an unverified unit ID |
| Siemens | `modbus-tcp` | Yes | `127.0.0.1` | `1514` | No | `[1]` | No | `8 / none / 1` | N/A for Modbus TCP | Partial | The SINAMICS app note supports the Modbus TCP protocol and register areas, but not the current simulator port or unit ID |

## Runtime Framing

- The Go server currently accepts only Modbus TCP MBAP frames with `Protocol Identifier = 0`.
- This is enforced in `internal/modbus/server.go`, not in the vendor JSON files.
- The JSON fields `data_bits`, `parity`, and `stop_bits` are metadata only in the current runtime and should not be read as active TCP transport settings.

## Audit Summary

- `Delta` is the most complete folder and still has the strongest documentation-to-runtime alignment.
- `ABB`, `Danfoss`, `Invertek`, `Schneider`, and `Siemens` all have configs and basic runtime behavior, but they are still partial implementations.
- `Omron` now has a partial MX2 config based on the documented flexible-format mapping, but it still lacks vendor-specific runtime behavior and verified network defaults.
- `Schneider` and `Siemens` still have official vendor documents identified online that could not be re-downloaded automatically because the vendor CDNs returned access-denied responses from this environment.
- No current simulator `port` value should be treated as vendor-document-verified unless that evidence is added explicitly to the vendor audit notes.
