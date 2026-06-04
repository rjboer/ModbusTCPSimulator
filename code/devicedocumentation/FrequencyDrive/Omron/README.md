# Omron Audit

## Local Documents Present

- `i666_mx2-ev2_users_manual_en.pdf`
- `i361_mx2_ethernet_ip_option_board_users_manual_en.pdf`
- `i114e_mx2_rx_componet_option_board_users_manual_en.pdf`
- `omron_mx2_drive_c.json`

## Official Documents Identified Online

- The  Omron files were recovered from official `assets.omron.eu` manual URLs.

## Documents Still Missing Locally

- No additional Omron Modbus TCP-specific document was recovered in this pass beyond the current drive and option-board manuals.

## Documentation Trust Status

- The local Omron PDFs are treated as the authoritative source set for this partial implementation.
- No third-party mirrored PDFs are used here.

## What Is Verified From Docs

- The folder contains official Omron source manuals for the MX2-V2 family and option-board behavior.
- The option-board manuals reference Modbus register addressing and configurable mapping behavior.
- `omron_mx2_drive_c.json` follows the typical flexible-format mapping described by the local Omron option-board manuals:
  - control via coils `0000h` through `000Fh`
  - status via coils `0010h` through `001Fh`
  - frequency reference via registers `0001h` and `0002h`
  - output frequency monitor `d001` via registers `1001h` and `1002h`

## What Remains Unverified

- No Omron network defaults for this simulator have been derived into JSON.
- No Omron-specific runtime behavior is implemented in Go yet.
- Whether the current source set is sufficient for a complete Omron Modbus TCP mock still depends on the exact drive family and interface option to be emulated.
