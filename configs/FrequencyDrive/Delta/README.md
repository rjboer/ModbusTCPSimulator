# Delta Audit

## Local Documents Present

- `Delta-MS-300-Series-Users-Manual.pdf`
- `Delta-VFD-MS300-User-Manual.pdf`
- `DELTA_IA-MDS_VFD-MS300_UM_EN_20170306.pdf`
- `DELTA_IA-MDS_MS300_C_EN_20210714.pdf`
- `DELTA_IA-MDS_VFD_EtherNetIP_AM_EN_20250219.pdf`
- `delta_ms300_drive_c.json`

## Official Documents Identified Online

- Delta MS300 product pages identify the series as supporting optional communication card installation including `MODBUS TCP`.
- Delta product pages also identify the MS300 catalogue and related application material used to recover the local source set.

## Documents Still Missing Locally

- No Delta document was found in this pass that cleanly states a Modbus TCP default port or unit-ID policy for the MS300 profile used by this simulator.

## Documentation Trust Status

- The local Delta PDFs listed above are treated as the authoritative local source set.
- `DELTA_IA-MDS_MS300_C_EN_20210714.pdf` and `DELTA_IA-MDS_VFD-MS300_UM_EN_20170306.pdf` were recovered from Delta-hosted CDN links and verified as real PDFs.
- No third-party mirrored PDFs are used here for network validation.

## What Is Verified From Docs

- `delta_ms300_drive_c.json` is grounded in the Modbus appendix of `Delta-MS-300-Series-Users-Manual.pdf` and refined with `register-map-corrected-monitoring.csv`.
- The command area `2000H` to `2002H` is represented in the holding-register block with corrected descriptions from the local Delta material.
- The monitor and display area from `2100H` through the documented `2234H` entries is represented in the input-register block with corrected descriptions from the local Delta material.
- The corrected CSV also confirms the communication-related parameter addresses at `0014H`, `0015H`, `0900H` through `0957H`, and the `2000H` command decoding dependency on `Pr.09-30 = 0`.
- The broader Delta product documentation confirms that the MS300 platform supports `MODBUS TCP` via communication card options.
- This is the only profile that also has vendor-specific runtime synchronization in the Go code.

## What Remains Unverified

- The simulator `listen_address`, `port`, and `unit_ids: [1]` are local defaults, not Delta-documented network defaults.
- No Delta source in this pass provided a clean Modbus TCP default port statement for this simulator profile.
- `data_bits`, `parity`, and `stop_bits` are metadata only in the current runtime and are not validated as active TCP settings.
- Not every MS300 parameter family outside the documented command and monitor areas is modeled.
- The communication-setup and Ethernet-option parameters from the corrected CSV are documented, but they are not exposed as a second sparse holding-register block because the current config schema only supports one contiguous block per register type.
