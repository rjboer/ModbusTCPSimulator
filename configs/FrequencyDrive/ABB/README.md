# ABB Audit

## Local Documents Present

- `EN_FENA01_11_21_UM_E_A4.pdf`
- `abb_fena_drive_c.json`

## Official Documents Identified Online

- No additional ABB vendor-hosted document URLs were captured during this pass beyond the local FENA manual already in the folder.

## Documents Still Missing Locally

- No ABB drive-family-specific parameter manual was added for this mock.

## Documentation Trust Status

- `EN_FENA01_11_21_UM_E_A4.pdf` is treated as the authoritative local source for this folder.
- No third-party mirrored PDFs are used here for network validation.

## What Is Verified From Docs

- `abb_fena_drive_c.json` is based on the FENA Modbus/TCP register map in `EN_FENA01_11_21_UM_E_A4.pdf`.
- The JSON descriptions match the documented ABB profile registers `400001` to `400015` and `400051` to `400065`.
- The runtime stores raw Modbus register addresses `1` to `65`, and the JSON descriptions preserve the documented `4xxxxx` numbering.
- The profile is Modbus TCP capable.

## What Remains Unverified

- The simulator `listen_address` and `port` are local defaults, not ABB-documented settings.
- The configured `unit_ids: [1]` is not confirmed from the ABB source set.
- `data_bits`, `parity`, and `stop_bits` are metadata only in the current runtime and are not validated as active TCP settings.
- Registers `16` to `50` remain placeholders in the JSON.
- This config models the FENA adapter profile, not a specific ABB drive family parameter set.
