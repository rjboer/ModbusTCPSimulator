# Danfoss Audit

## Local Documents Present

- `AQ361179371336en-000201.pdf`
- `AU275636650261en-003001.pdf`
- `AU554166676251en-000101.pdf`
- `AN542940142245en-000101.pdf`
- `danfoss_fc302_drive_c.json`

## Official Documents Identified Online

- No additional Danfoss vendor-hosted document URLs were added during this pass. The local PDF set remains the working source set.

## Documents Still Missing Locally

- No broader FC302 parameter reference was added beyond the local PDFs already present.

## Documentation Trust Status

- The local Danfoss PDFs are treated as the authoritative source set for this folder.
- No third-party mirrored PDFs are used here for network validation.

## What Is Verified From Docs

- `danfoss_fc302_drive_c.json` was built from the Modbus TCP process-data documentation in `AQ361179371336en-000201.pdf`.
- The documented cyclic write block `2810` to `2819` and read block `2910` to `2919` are represented in the JSON.
- The layout of the first words as control/reference and status/actual value matches the manual-backed process-data block.
- The profile is Modbus TCP capable.
- The current config keeps `unit_ids: [255]` because that value was used in the prior manual-backed audit for this profile.

## What Remains Unverified

- The simulator `listen_address` and `port` are local defaults, not Danfoss-documented settings.
- This pass did not re-extract the exact unit-ID wording from the PDFs, so the current `unit_ids` entry should be treated as previously audited rather than newly re-verified.
- `data_bits`, `parity`, and `stop_bits` are metadata only in the current runtime and are not validated as active TCP settings.
- The JSON does not model the full FC302 parameter register space.
- Large parts of the contiguous block remain placeholders between the documented cyclic areas.
