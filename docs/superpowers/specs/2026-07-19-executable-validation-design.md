# Executable Validation Design

## Goal

Validate the Windows deliverables for the Modbus TCP Simulator after the security dependency updates, using the repository's existing build workflow and the machine's working 64-bit MinGW compiler.

## Findings

- `cmd/fynehmi` builds the GUI executable and requires CGO.
- `cmd/mockserver` builds the console/server executable.
- `build.ps1` already supports `-Target all`, `-OutputDir`, `-CCPath`, and `-CXXPath`.
- The first `gcc` on PATH is a 32-bit Simcenter compiler (`mingw32`) and causes the observed 64-bit CGO failure.
- `C:\TDM-GCC-64\bin\gcc.exe` reports `x86_64-w64-mingw32` and is the valid compiler.
- PowerShell's execution policy blocks direct script invocation, so the script must be launched with process-scoped `ExecutionPolicy Bypass`.

## Execution

Run the existing build script with the explicit compiler paths and write outputs to `bin\\validated` so existing binaries are preserved. Verify the output files, PE metadata, and the console executable's safe `--help` path. Run the repository's backend checks separately; GUI runtime launch is not attempted because it requires an interactive desktop session.

## Success Criteria

- Both fresh executables are produced under `bin\\validated`.
- The build exits successfully with the 64-bit compiler.
- `mockserver.exe --help` exits successfully without starting a server.
- `go test ./internal/app ./internal/modbus` passes.
- No source files are changed by this validation.
