param(
    [ValidateSet("fynehmi", "mockserver", "all")]
    [string]$Target = "fynehmi",
    [string]$OutputDir = "bin",
    [switch]$Bundle,
    [string]$BundleDir = "dist\modbus-tcp-simulator-windows",
    [string]$CCPath = "",
    [string]$CXXPath = ""
)

$ErrorActionPreference = "Stop"

function Test-MingwCompiler {
    param(
        [string]$CompilerPath
    )

    if (-not (Test-Path -LiteralPath $CompilerPath -PathType Leaf)) {
        return $false
    }

    $machine = & $CompilerPath -dumpmachine 2>$null
    return $LASTEXITCODE -eq 0 -and $machine -match "^x86_64.*mingw"
}

function Resolve-MingwCompiler {
    param(
        [string]$RequestedPath,
        [string[]]$CandidatePaths
    )

    if ($RequestedPath) {
        if (-not (Test-Path -LiteralPath $RequestedPath -PathType Leaf)) {
            throw "Requested compiler not found: $RequestedPath"
        }

        $resolvedRequestedPath = (Resolve-Path -LiteralPath $RequestedPath).Path
        if (-not (Test-MingwCompiler -CompilerPath $resolvedRequestedPath)) {
            throw "Requested compiler does not target 64-bit MinGW: $resolvedRequestedPath"
        }
        return $resolvedRequestedPath
    }

    foreach ($candidate in $CandidatePaths) {
        if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
            continue
        }

        $resolvedCandidate = (Resolve-Path -LiteralPath $candidate).Path
        if (Test-MingwCompiler -CompilerPath $resolvedCandidate) {
            return $resolvedCandidate
        }
    }

    $whereGcc = & where.exe gcc 2>$null
    foreach ($gcc in $whereGcc) {
        if (Test-MingwCompiler -CompilerPath $gcc) {
            return (Resolve-Path -LiteralPath $gcc).Path
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
        if (-not (Test-Path -LiteralPath $RequestedPath -PathType Leaf)) {
            throw "Requested C++ compiler not found: $RequestedPath"
        }

        $resolvedRequestedPath = (Resolve-Path -LiteralPath $RequestedPath).Path
        if (-not (Test-MingwCompiler -CompilerPath $resolvedRequestedPath)) {
            throw "Requested C++ compiler does not target 64-bit MinGW: $resolvedRequestedPath"
        }
        return $resolvedRequestedPath
    }

    $candidate = $ResolvedCC -replace "gcc\.exe$", "g++.exe"
    if (Test-MingwCompiler -CompilerPath $candidate) {
        return (Resolve-Path -LiteralPath $candidate).Path
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
        "C:\TDM-GCC-64\bin\gcc.exe",
        "C:\msys64\ucrt64\bin\gcc.exe",
        "C:\msys64\mingw64\bin\gcc.exe"
    )

$resolvedCXX = Resolve-CxxCompiler -RequestedPath $CXXPath -ResolvedCC $resolvedCC

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:GOAMD64 = "v1"
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

if ($Bundle) {
    $resolvedBundleDir = Join-Path (Get-Location) $BundleDir
    New-Item -ItemType Directory -Force -Path $resolvedBundleDir | Out-Null

    if ($Target -eq "fynehmi" -or $Target -eq "all") {
        Copy-Item -LiteralPath (Join-Path $resolvedOutputDir "fynehmi.exe") -Destination $resolvedBundleDir -Force
    }
    if ($Target -eq "mockserver" -or $Target -eq "all") {
        Copy-Item -LiteralPath (Join-Path $resolvedOutputDir "mockserver.exe") -Destination $resolvedBundleDir -Force
    }

    Copy-Item -Recurse -LiteralPath (Join-Path (Get-Location) "configs") -Destination (Join-Path $resolvedBundleDir "configs") -Force
    if (Test-Path (Join-Path (Get-Location) "Readme.md")) {
        Copy-Item -LiteralPath (Join-Path (Get-Location) "Readme.md") -Destination $resolvedBundleDir -Force
    }
}

Write-Host ""
Write-Host "Build completed."
