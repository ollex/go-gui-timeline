param(
    [ValidateRange(1, 65535)]
    [int]$Port = 8080
)

$ErrorActionPreference = 'Stop'
$projectRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..')).Path
$outputDir = Join-Path $projectRoot 'build\timeline-web'
$wasmPath = Join-Path $outputDir 'timeline.wasm'
$goRoot = (& go env GOROOT).Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($goRoot)) {
    throw 'Could not determine GOROOT.'
}
$wasmExec = Join-Path $goRoot 'lib\wasm\wasm_exec.js'
if (-not (Test-Path -LiteralPath $wasmExec)) {
    throw "Could not find $wasmExec"
}

New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'web\index.html') -Destination $outputDir -Force
Copy-Item -LiteralPath $wasmExec -Destination $outputDir -Force

$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
Push-Location $projectRoot
try {
    $env:GOOS = 'js'
    $env:GOARCH = 'wasm'
    & go build -o $wasmPath '.\examples\timeline'
    if ($LASTEXITCODE -ne 0) {
        throw 'The WebAssembly build failed.'
    }
} finally {
    if ($null -eq $previousGOOS) {
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    } else {
        $env:GOOS = $previousGOOS
    }
    if ($null -eq $previousGOARCH) {
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    } else {
        $env:GOARCH = $previousGOARCH
    }
}

try {
    & go run '.\examples\timeline\webserver' "-addr=127.0.0.1:$Port" "-dir=$outputDir"
    if ($LASTEXITCODE -ne 0) {
        throw 'The local web server stopped with an error.'
    }
} finally {
    Pop-Location
}
