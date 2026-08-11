# Executable Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build and safely validate both Windows executables using the repository's existing build script and its available 64-bit MinGW compiler.

**Architecture:** Keep the existing PowerShell build workflow unchanged. Override only the compiler selection and PowerShell execution policy for this validation run, writing artifacts to an isolated output directory. Validate the console executable non-interactively and use static metadata for the GUI executable.

**Tech Stack:** Go 1.26.5, PowerShell, `build.ps1`, TDM-GCC-64, Windows PE executables.

## Global Constraints

- Use `C:\TDM-GCC-64\bin\gcc.exe` and `C:\TDM-GCC-64\bin\g++.exe`.
- Use `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\build.ps1`.
- Write fresh artifacts only to `bin\\validated`.
- Do not launch the GUI executable automatically.
- Do not alter source files or existing binaries.

### Task 1: Build both Windows executables

**Files:**
- Read: `build.ps1`
- Output: `bin\\validated\\fynehmi.exe`, `bin\\validated\\mockserver.exe`

**Interfaces:**
- Consumes: `build.ps1 -Target all -OutputDir bin\\validated -CCPath C:\\TDM-GCC-64\\bin\\gcc.exe -CXXPath C:\\TDM-GCC-64\\bin\\g++.exe`
- Produces: two fresh Windows PE executables.

- [ ] **Step 1: Run the existing build script with the explicit 64-bit compiler**

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\build.ps1 `
  -Target all `
  -OutputDir .\bin\validated `
  -CCPath C:\TDM-GCC-64\bin\gcc.exe `
  -CXXPath C:\TDM-GCC-64\bin\g++.exe
```

Expected: exit code 0 and `Build completed.`.

- [ ] **Step 2: Verify both output files exist and are fresh**

```powershell
Get-Item .\bin\validated\fynehmi.exe, .\bin\validated\mockserver.exe |
  Select-Object FullName, Length, LastWriteTime
```

Expected: both files exist with non-zero sizes and current timestamps.

### Task 2: Smoke-test the produced executables

**Files:**
- Read: `bin\\validated\\fynehmi.exe`, `bin\\validated\\mockserver.exe`

**Interfaces:**
- Consumes: the two executables from Task 1.
- Produces: process exit evidence and PE metadata.

- [ ] **Step 1: Inspect PE metadata**

```powershell
Get-Command .\bin\validated\fynehmi.exe, .\bin\validated\mockserver.exe |
  Select-Object Source, FileVersionInfo
```

Expected: both resolve as Windows executables.

- [ ] **Step 2: Exercise the non-interactive help path**

```powershell
& .\bin\validated\mockserver.exe --help
if ($LASTEXITCODE -ne 0) { throw "mockserver --help failed with exit code $LASTEXITCODE" }
```

Expected: usage text and exit code 0; no listener is started.

- [ ] **Step 3: Do not launch the GUI automatically**

The GUI executable requires an interactive desktop and is validated by successful compilation plus PE metadata inspection.

### Task 3: Run repository checks and confirm source cleanliness

**Files:**
- Read: `go.mod`, `go.sum`, repository source tree.

**Interfaces:**
- Consumes: the current dependency graph and source tree.
- Produces: test output and a clean-source status.

- [ ] **Step 1: Run backend checks**

```powershell
go test ./internal/app ./internal/modbus
```

Expected: exit code 0.

- [ ] **Step 2: Confirm no source changes**

```powershell
git status --short
```

Expected: no modified or untracked source files; only the explicitly created Superpowers documentation may appear as new files.
