param(
    [string]$CCPath = "",
    [string]$CXXPath = ""
)

$ErrorActionPreference = "Stop"

function Test-MingwCompiler {
    param([string]$CompilerPath)
    if (-not (Test-Path -LiteralPath $CompilerPath -PathType Leaf)) { return $false }
    $machine = & $CompilerPath -dumpmachine 2>$null
    return $LASTEXITCODE -eq 0 -and $machine -match "^x86_64.*mingw"
}

$candidates = @(
    "C:\TDM-GCC-64\bin\gcc.exe",
    "C:\msys64\ucrt64\bin\gcc.exe",
    "C:\msys64\mingw64\bin\gcc.exe"
)

if ([string]::IsNullOrWhiteSpace($CCPath)) {
    foreach ($candidate in $candidates) {
        if (Test-MingwCompiler $candidate) {
            $CCPath = $candidate
            break
        }
    }
}
if ([string]::IsNullOrWhiteSpace($CCPath) -or -not (Test-MingwCompiler $CCPath)) {
    throw "No 64-bit MinGW gcc.exe found. Pass -CCPath explicitly."
}
if ([string]::IsNullOrWhiteSpace($CXXPath)) {
    $CXXPath = Join-Path (Split-Path -Parent $CCPath) "g++.exe"
}
if (-not (Test-MingwCompiler $CXXPath)) {
    throw "C++ compiler is missing or does not target 64-bit MinGW: $CXXPath"
}

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:GOAMD64 = "v1"
$env:CGO_ENABLED = "1"
$env:CC = $CCPath
$env:CXX = $CXXPath
$env:PATH = "$(Split-Path -Parent $CCPath);$env:PATH"

$ldPath = Join-Path (Split-Path -Parent $CCPath) "ld.exe"
$versionOutput = @(& $ldPath --version 2>&1)
$firstLine = [string]$versionOutput[0]
if ($firstLine -match "([0-9]+)[.]([0-9]+)") {
    if ([version]"$($Matches[1]).$($Matches[2])" -lt [version]"2.37") {
        $env:GOEXPERIMENT = "nodwarf5"
    }
}

go test -mod=readonly ./... -count=1
if ($LASTEXITCODE -ne 0) { throw "go test failed" }
