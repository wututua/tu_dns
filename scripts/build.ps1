$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

Push-Location (Join-Path $root "frontend")
try {
    npm ci
    if ($LASTEXITCODE -ne 0) { throw "npm ci failed" }
    npm run build
    if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }
}
finally {
    Pop-Location
}

$bin = Join-Path $root "bin"
New-Item -ItemType Directory -Force -Path $bin | Out-Null
Push-Location $root
try {
    go build -trimpath -o (Join-Path $bin "tudns.exe") .
    if ($LASTEXITCODE -ne 0) { throw "Go build failed" }
}
finally {
    Pop-Location
}
