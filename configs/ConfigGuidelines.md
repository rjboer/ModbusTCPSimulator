# Configuration Guidelines

This folder contains configuration guidance for Modbus TCP mock devices used by the server.

## File Naming

Discoverable configuration files must end with `c.json`.

Recommended naming pattern:

- `{device_name}_{device_type}_c.json`

Examples:

- `delta_ms300_drive_c.json`
- `testbench_pump_drive_c.json`
- `siemens_demo_device_c.json`

Files that do not end with `c.json` are ignored by the launcher during config discovery.

## Required Top-Level Fields

Each config should contain at least:

- `name`
- `device_type`
- `device_profile`
- `listen_address`
- `port`
- `unit_ids`
- `log_level`
- `network`
- `connection`
- `documentation` when the file should carry audit or provenance information
- `runtime` when the mock should derive live values from command registers
- `data_model`

## Field Intent

- `name`: human-readable device name shown in the launcher menu and register map view
- `device_type`: broad category such as `frequency-drive` or `generic-modbus-device`
- `device_profile`: mock behaviour profile such as `delta-ms300`
- `listen_address`: server bind address
- `port`: Modbus TCP port
- `unit_ids`: allowed Modbus unit ids
- `log_level`: current log verbosity field reserved for runtime filtering

## Network Section

The `network` section stores protocol and framing metadata with the device definition even though the current runtime serves Modbus TCP only.

Recommended fields:

- `protocol`
- `data_bits`
- `parity`
- `stop_bits`

Example:

```json
"network": {
  "protocol": "modbus-tcp",
  "data_bits": 8,
  "parity": "none",
  "stop_bits": 1
}
```

## Connection Section

The `connection` section defines server-side operating limits.

Example:

```json
"connection": {
  "max_clients": 8,
  "idle_timeout_ms": 30000
}
```

## Documentation Section

The `documentation` section is optional metadata for audit provenance. It does not affect the runtime behavior of the server, but it is the preferred place to record which network fields are vendor-documented and which are simulator defaults.

Recommended fields:

- `trust`
- `local_sources`
- `official_references`
- `network_validation`
- `notes`

Example:

```json
"documentation": {
  "trust": "partial",
  "local_sources": [
    "Modbus_TCP_SINAMICS_V1_0_en.pdf"
  ],
  "official_references": [
    "https://support.industry.siemens.com/cs/attachments/109812989/SINAMICS_DCP_en-US.pdf"
  ],
  "network_validation": {
    "protocol": "documented",
    "listen_address": "simulator-default",
    "port": "simulator-default",
    "unit_ids": "unverified",
    "protocol_id": "runtime-enforced-zero",
    "serial_framing": "metadata-only"
  },
  "notes": [
    "The current simulator port is a local test choice, not a vendor-documented default."
  ]
}
```

## Data Model Section

The `data_model` section defines the exposed Modbus blocks:

- `coils`
- `discrete_inputs`
- `holding_registers`
- `input_registers`

Each block should define:

- `start_address`
- `values`
- `descriptions` when the values should be self-explanatory in the runtime register map

The `descriptions` list should have the same number of items as the `values` list for that block.

Example:

```json
"holding_registers": {
  "start_address": 8192,
  "values": [0, 5000, 0],
  "descriptions": [
    "Operation command",
    "Frequency command in 0.01 Hz units",
    "Fault and control command source"
  ]
}
```

## Runtime Behavior Section

The `runtime` section is optional, but it is the recommended place to describe how writable registers affect readback registers.

This keeps the address binding in JSON instead of hardcoding the addresses in Go.

Current supported model:

- `abb-fena-basic`
- `danfoss-fc302-basic`
- `delta-ms300-basic`
- `invertek-optidrive-p2-basic`
- `schneider-atv320-basic`
- `siemens-sinamics-basic`

Recommended fields:

- `runtime.behavior.model`
- `runtime.behavior.register_map`
- `runtime.behavior.constants`

Example:

```json
"runtime": {
  "behavior": {
    "model": "delta-ms300-basic",
    "register_map": {
      "command": 8192,
      "frequency_command": 8193,
      "drive_operation_status": 8449,
      "output_frequency": 8451,
      "drive_status": 8742
    },
    "constants": {
      "stopped_status_word": 1280,
      "running_forward_drive_status": 21,
      "running_reverse_drive_status": 22
    }
  }
}
```

Notes:

- `register_map` binds semantic names such as `command`, `output_frequency`, or `drive_status` to concrete Modbus addresses in the config.
- `constants` carries fixed values or masks used by the runtime model.
- The Go runtime still owns the behavior algorithm. The config owns the address map and constants.
- This is the intended extension point for adding ABB, Danfoss, Siemens, and Schneider runtime behavior later without hardcoding each register address.

## Device Folders

Each branded device family should have its own folder under `configs`.

That folder should contain at least:

- a `README.md`
- one or more example `*c.json` configs
- device-specific notes as needed
