# Schneider Audit

## Local Documents Present

- `ATV320_CommunicationParameters_NVE41316_V3.5.xlsx`
- `A700000009065926.pdf`
- `ATV320_Modbus_manual_EN_NVE41308_01.pdf`
- `ATV320_Modbus_TCP_EtherNet_IP_Manual_NVE41313_02.pdf`
- `schneider_atv320_drive_c.json`

## Official Documents Identified Online

- `https://www.se.com/pl/pl/faqs/FA311640/`
  - identifies `ATV320_Modbus_manual_EN_NVE41308_01.pdf`
  - identifies `ATV320_communication_parameters_NVE41316_V2.7.xls.7z`
- `https://www.se.com/uk/en/download/document/ATV320_additional_EN/`
  - lists `ATV320_Modbus_manual_EN_NVE41308_01.pdf`
  - lists `ATV320_Modbus_TCP_EtherNet_IP_Manual_NVE41313_02.pdf`
- `https://www.se.com/us/en/download/document/NVE41313/`
  - official Schneider download page for `ATV320_Modbus_TCP_EtherNet_IP_Manual_NVE41313_02.pdf`

## Documents Still Missing Locally

- No additional Schneider communication document is currently required for the existing partial ATV320 mock profile.

## Documentation Trust Status

- `ATV320_CommunicationParameters_NVE41316_V3.5.xlsx` is treated as the authoritative local source for the current register-map audit.
- `ATV320_Modbus_manual_EN_NVE41308_01.pdf` is now present locally and can be used as a local source for the ATV320 Modbus serial-link documentation set.
- `ATV320_Modbus_TCP_EtherNet_IP_Manual_NVE41313_02.pdf` is also present locally and can be used as a local source for the ATV320 Modbus TCP / EtherNet/IP documentation set.
- The official Schneider pages above are treated as authoritative references for document existence and filenames.
- The official Schneider CDN still returned access-denied responses during scripted download attempts in this environment, so provenance of the local files should be treated as local-user-supplied rather than directly re-downloaded by the agent.

## What Is Verified From Docs

- `schneider_atv320_drive_c.json` was checked mainly against `ATV320_CommunicationParameters_NVE41316_V3.5.xlsx`.
- The JSON includes the documented logic addresses for:
  - `CMD` at `8501`
  - `LFR` at `8502`
  - `PISP` at `8503`
  - `CMI` at `8504`
  - `LFRD` at `8602`
  - `RFRD` at `8604`
  - `ETA` at `3201`
  - `RFR` at `3202`
  - `LCR` at `3204`
  - `OTR` at `3205`
  - `ETI` at `3206`
  - `UOP` at `3208`
  - `OPR` at `3211`
- Schneider's official document pages confirm that both the serial Modbus manual `NVE41308` and the Modbus TCP / EtherNet/IP manual `NVE41313` exist for the ATV320 family.
- The local folder now contains both referenced Schneider manuals, so the documentation set for the current ATV320 audit is locally complete.

## What Remains Unverified

- The simulator `listen_address`, `port`, and `unit_ids: [1]` are local defaults, not Schneider-documented network defaults.
- `data_bits`, `parity`, and `stop_bits` are metadata only in the current runtime and are not validated as active TCP settings.
- Several documented workbook addresses are still missing from the JSON, including examples such as `HMIS 3240`, `AIV1 5281`, `CNFS 8020`, `CRC 8441`, `CCC 8442`, `STUN 9617`, and `SMOT 9645`.
- The local environment received Schneider CDN access-denied responses when attempting direct scripted download from the official attachment paths, so the agent could not independently re-fetch the files from Schneider during the audit.
