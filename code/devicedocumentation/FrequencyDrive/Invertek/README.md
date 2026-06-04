# Invertek Audit

## Local Documents Present

- `Optidrive_P2_Advanced_User_Guide_Rev_2.00.pdf`
- `Quick_Start_Guide_Modbus_TCP_Interface.pdf`
- `414769.pdf`
- `invertek_optidrive_p2_drive_c.json`

## Official Documents Identified Online

- Invertek product pages for Optidrive P2 options identify:
  - `OPT-2-MODIP-IN` Modbus TCP Module
  - `OPT-3-MTPEG-IN` Modbus TCP Interface
- Invertek's communication-interface page explicitly shows the `User Guide & Config Files` column as `–` for `OPT-2-MODIP-IN`.

## Documents Still Missing Locally

- No vendor-hosted `OPT-2-MODIP-IN` user guide or config file was found during this pass.
- No vendor-hosted document was recovered that states a Modbus TCP default port or unit-ID policy for this simulator profile.

## Documentation Trust Status

- The local Invertek PDFs remain the authoritative local source set for register mapping.
- The official Invertek product pages are used as authoritative evidence for what modules exist and for the absence of a published `OPT-2-MODIP-IN` user guide in the page table.
- No third-party mirrored PDFs are used here for network validation.

## What Is Verified From Docs

- `invertek_optidrive_p2_drive_c.json` was checked against the documented Modbus TCP cyclic registers in `Optidrive_P2_Advanced_User_Guide_Rev_2.00.pdf`.
- The JSON maps the documented master-to-drive words `1` to `4`.
- The JSON maps the documented drive-to-master words `256` to `259`.
- The official Invertek product pages confirm Modbus TCP interface options exist for the P2 family.

## What Remains Unverified

- The simulator `listen_address`, `port`, and `unit_ids: [1]` are local defaults, not Invertek-documented network defaults.
- `data_bits`, `parity`, and `stop_bits` are metadata only in the current runtime and are not validated as active TCP settings.
- No broader Optidrive parameter map beyond the cyclic exchange is implemented.
- No vendor-hosted `OPT-2-MODIP-IN` user guide was recovered in this pass.
