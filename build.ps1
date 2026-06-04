param(
    [ValidateSet("fynehmi", "mockserver", "all")]
    [string]$Target = "fynehmi",
    [string]$OutputDir = "bin",
    [string]$CCPath = "",
    [string]$CXXPath = ""
)

$ErrorActionPreference = "Stop"

function Resolve-MingwCompiler {
    param(
        [string]$RequestedPath,
        [string[]]$CandidatePaths
    )

    if ($RequestedPath) {
        if (-not (Test-Path $RequestedPath)) {
            throw "Requested compiler not found: $RequestedPath"
        }
        return (Resolve-Path $RequestedPath).Path
    }

    foreach ($candidate in $CandidatePaths) {
        if (Test-Path $candidate) {
            return (Resolve-Path $candidate).Path
        }
    }

    $whereGcc = & where.exe gcc 2>$null
    foreach ($gcc in $whereGcc) {
        if (-not (Test-Path $gcc)) {
            continue
        }

        $machine = & $gcc -dumpmachine 2>$null
        if ($LASTEXITCODE -eq 0 -and $machine -match "x86_64.*mingw") {
            return $gcc
        }
    }

    throw "No 64-bit MinGW gcc compiler found. Install MSYS2/UCRT64 or TDM-GCC-64, or pass -CCPath."
}

function Resolve-CxxCompiler {
    param(
        [string]$RequestedPath,
        [string]$ResolvedCC
    )

    if ($RequestedPath) {
        if (-not (Test-Path $RequestedPath)) {
            throw "Requested C++ compiler not found: $RequestedPath"
        }
        return (Resolve-Path $RequestedPath).Path
    }

    $candidate = $ResolvedCC -replace "gcc\.exe$", "g++.exe"
    if (Test-Path $candidate) {
        return (Resolve-Path $candidate).Path
    }

    throw "Could not infer g++.exe from $ResolvedCC. Pass -CXXPath explicitly."
}

function Invoke-GoBuild {
    param(
        [string]$Package,
        [string]$Output
    )

    Write-Host "Building $Package -> $Output"
    & go build -o $Output $Package
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed for $Package"
    }
}

$resolvedCC = Resolve-MingwCompiler `
    -RequestedPath $CCPath `
    -CandidatePaths @(
        "C:\msys64\ucrt64\bin\gcc.exe",
        "C:\msys64\mingw64\bin\gcc.exe",
        "C:\TDM-GCC-64\bin\gcc.exe"
    )

$resolvedCXX = Resolve-CxxCompiler -RequestedPath $CXXPath -ResolvedCC $resolvedCC

$env:CGO_ENABLED = "1"
$env:CC = $resolvedCC
$env:CXX = $resolvedCXX

$resolvedOutputDir = Join-Path (Get-Location) $OutputDir
New-Item -ItemType Directory -Force -Path $resolvedOutputDir | Out-Null

Write-Host "Using compiler:"
Write-Host "  CC  = $env:CC"
Write-Host "  CXX = $env:CXX"
Write-Host "  GO  = $(go version)"
Write-Host ""

switch ($Target) {
    "fynehmi" {
        Invoke-GoBuild -Package "./cmd/fynehmi" -Output (Join-Path $resolvedOutputDir "fynehmi.exe")
    }
    "mockserver" {
        Invoke-GoBuild -Package "./cmd/mockserver" -Output (Join-Path $resolvedOutputDir "mockserver.exe")
    }
    "all" {
        Invoke-GoBuild -Package "./cmd/fynehmi" -Output (Join-Path $resolvedOutputDir "fynehmi.exe")
        Invoke-GoBuild -Package "./cmd/mockserver" -Output (Join-Path $resolvedOutputDir "mockserver.exe")
    }
}

Write-Host ""
Write-Host "Build completed."
