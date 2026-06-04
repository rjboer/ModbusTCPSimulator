# Siemens Audit

## Local Documents Present

- `Modbus_TCP_SINAMICS_V1_0_en.pdf`
- `siemens_sinamics_drive_c.json`

## Official Documents Identified Online

- `https://support.industry.siemens.com/cs/attachments/109812989/SINAMICS_DCP_en-US.pdf`
- `https://support.industry.siemens.com/cs/attachments/99005531/Manual-dcp-en.pdf`

## Documents Still Missing Locally

- No additional Siemens vendor-hosted PDF was recovered locally during this pass beyond `Modbus_TCP_SINAMICS_V1_0_en.pdf`.
- Broader SINAMICS DCP / network-behavior documents remain online-identified but not locally recovered.

## Documentation Trust Status

- `Modbus_TCP_SINAMICS_V1_0_en.pdf` remains the authoritative local source for the current register-map audit.
- The Siemens support attachment URLs above are treated as authoritative official references, but not as local evidence because scripted download returned access-denied responses in this environment.
- No third-party mirrored PDFs are used here for network validation.

## What Is Verified From Docs

- `siemens_sinamics_drive_c.json` was checked against the SINAMICS Modbus TCP application note.
- The JSON includes the documented process-data write area `40100` to `40109`.
- The JSON includes the documented process-data read area `40110` to `40119`.
- The JSON also includes the documented standalone registers:
  - `40324` reference speed
  - `40400` failure number
  - `40408` alarm number
  - `40409` actual alarm code
- The profile is Modbus TCP capable.

## What Remains Unverified

- The simulator `listen_address`, `port`, and `unit_ids: [1]` are local defaults, not Siemens-documented network defaults.
- `data_bits`, `parity`, and `stop_bits` are metadata only in the current runtime and are not validated as active TCP settings.
- The JSON does not model the full SINAMICS parameter space.
- Large parts of the contiguous block remain placeholders.
- The broader Siemens support documents identified online could not be fetched locally because the Siemens CDN returned access-denied responses in this environment.
