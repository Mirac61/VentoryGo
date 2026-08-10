# start-dev.ps1
#
# Startet Backend, Frontend, Landing Page und den lokalen Dev-Proxy in vier
# eigenen, sichtbaren PowerShell-Fenstern. Liegt im Repo-Root
# (VentoryGo\start-dev.ps1), neben backend/, frontend/, landing/, dev-proxy/.
#
# Voraussetzungen (einmalig, nicht Teil dieses Skripts):
#   - Docker Desktop laeuft bereits
#   - go, npm, node sind auf PATH
#   - backend/.env existiert und enthaelt DATABASE_URL
#
# Nutzung:
#   .\start-dev.ps1

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot

function Read-DotEnvValue {
    param([string]$Path, [string]$Key)

    if (-not (Test-Path $Path)) {
        throw "Datei nicht gefunden: $Path"
    }

    $line = Get-Content $Path | Where-Object { $_ -match "^\s*$Key\s*=" } | Select-Object -First 1
    if (-not $line) {
        throw "$Key nicht gefunden in $Path"
    }

    return ($line -split '=', 2)[1].Trim()
}

Write-Host "Lese DATABASE_URL aus backend\.env ..." -ForegroundColor Cyan
$databaseUrl = Read-DotEnvValue -Path (Join-Path $root 'backend\.env') -Key 'DATABASE_URL'

Write-Host "Starte Postgres-Container ..." -ForegroundColor Cyan
Push-Location (Join-Path $root 'backend')
docker compose up -d postgres
Pop-Location

Write-Host "Warte kurz, bis Postgres bereit ist ..." -ForegroundColor Cyan
Start-Sleep -Seconds 3

# --- Backend ---
Write-Host "Starte Backend (Fenster 1) ..." -ForegroundColor Green
Start-Process powershell -ArgumentList @(
    '-NoExit',
    '-Command',
    "cd '$root\backend'; `$env:DATABASE_URL = '$databaseUrl'; go run ."
)

# --- Frontend ---
Write-Host "Starte Frontend (Fenster 2) ..." -ForegroundColor Green
Start-Process powershell -ArgumentList @(
    '-NoExit',
    '-Command',
    "cd '$root\frontend'; npm run dev"
)

# --- Landing Page ---
# WICHTIG: serve laeuft im Repo-Root (nicht landing/), sonst kann die Seite
# nicht auf shared/fonts/ zugreifen. Der Proxy schreibt "/" intern auf
# /landing/index.html um.
Write-Host "Starte Landing Page (Fenster 3) ..." -ForegroundColor Green
Start-Process powershell -ArgumentList @(
    '-NoExit',
    '-Command',
    "cd '$root'; npx serve . -l 4173"
)

# --- Dev-Proxy ---
Start-Sleep -Seconds 2
Write-Host "Starte Dev-Proxy (Fenster 4) ..." -ForegroundColor Green
Start-Process powershell -ArgumentList @(
    '-NoExit',
    '-Command',
    "cd '$root\dev-proxy'; node proxy.js"
)

Write-Host ""
Write-Host "Alle vier Fenster gestartet. Alles laeuft unter http://localhost:8090" -ForegroundColor Yellow
Write-Host "Fenster einzeln schliessen, um den jeweiligen Prozess zu beenden." -ForegroundColor Yellow