#Requires -Version 7.0
<#
.SYNOPSIS
    Startet den kompletten Dev-Stack: Postgres, Migrationen, Backend, Frontend, Landing Page, Dev-Proxy.
.DESCRIPTION
    Windows-Gegenstück zu dev.sh. Beide Skripte machen dasselbe in derselben
    Reihenfolge; Änderungen an einem gehören ins andere.
#>

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$backendPort = 8080
$frontendPort = 5173
$landingPort = 4173
$proxyPort = 8090

function Write-Log($message) { Write-Host "[dev] $message" -ForegroundColor Green }
function Stop-WithError($message) { Write-Host "[dev] $message" -ForegroundColor Red; exit 1 }

# --- Preflight ---------------------------------------------------------------

function Assert-Command($name, $hint) {
    if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
        Stop-WithError "$name nicht gefunden. $hint"
    }
}

function Assert-File($path, $hint) {
    if (-not (Test-Path $path)) {
        Stop-WithError "$path fehlt. Anlegen mit: $hint"
    }
}

function Test-PortInUse($port) {
    $client = New-Object System.Net.Sockets.TcpClient
    try {
        $client.Connect('127.0.0.1', $port)
        return $true
    } catch {
        return $false
    } finally {
        $client.Dispose()
    }
}

Assert-Command docker 'Docker Desktop starten.'
Assert-Command go 'Go installieren: https://go.dev/dl/'
Assert-Command npm 'Node installieren: https://nodejs.org/'
Assert-Command node 'Node installieren: https://nodejs.org/'

Assert-File '.env' 'copy .env.example .env'
Assert-File 'backend\.env' 'copy backend\.env.example backend\.env'

if (Test-PortInUse $backendPort) {
    Stop-WithError "Port $backendPort ist belegt. Läuft noch ein altes Backend?"
}
if (Test-PortInUse $frontendPort) {
    Write-Log "Port $frontendPort ist belegt - Vite weicht auf einen anderen aus."
}
if (Test-PortInUse $landingPort) {
    Stop-WithError "Port $landingPort ist belegt. Läuft noch eine alte Landing Page?"
}
if (Test-PortInUse $proxyPort) {
    Stop-WithError "Port $proxyPort ist belegt. Läuft noch ein alter Dev-Proxy?"
}

# `go run .` liest keine .env: main.go ruft nur os.Getenv. Ohne das hier fällt pgx auf
# die libpq-Defaults zurück und meldet einen Verbindungsfehler, der nach einem kaputten
# Postgres aussieht.
$backendEnv = @{}
foreach ($line in Get-Content 'backend\.env') {
    $trimmed = $line.Trim()
    if ($trimmed -eq '' -or $trimmed.StartsWith('#')) { continue }

    $separator = $trimmed.IndexOf('=')
    if ($separator -lt 0) { continue }

    $key = $trimmed.Substring(0, $separator).Trim()
    $value = $trimmed.Substring($separator + 1).Trim().Trim('"').Trim("'")
    $backendEnv[$key] = $value
    [Environment]::SetEnvironmentVariable($key, $value, 'Process')
}

# Eine vorhandene, aber unvollständige .env würde sonst erst bei der Migration auffallen -
# mit einer Meldung, die auf Postgres zeigt statt auf die Konfiguration.
if (-not $backendEnv['DATABASE_URL']) {
    Stop-WithError 'DATABASE_URL fehlt oder ist leer in backend\.env. Vorlage: backend\.env.example'
}

# --- Postgres ----------------------------------------------------------------

Write-Log 'Postgres starten ...'
docker compose up -d postgres
if ($LASTEXITCODE -ne 0) { Stop-WithError 'docker compose up -d postgres ist fehlgeschlagen.' }

$pgContainer = (docker compose ps -q postgres).Trim()
if (-not $pgContainer) { Stop-WithError 'Postgres-Container nicht gefunden.' }

Write-Log 'Warte, bis Postgres Verbindungen annimmt ...'
$waited = 0
while ((docker inspect --format '{{.State.Health.Status}}' $pgContainer).Trim() -ne 'healthy') {
    Start-Sleep -Seconds 1
    $waited++
    if ($waited -ge 60) {
        Stop-WithError 'Postgres wurde nicht healthy. Siehe: docker compose logs postgres'
    }
}

# --- Migrationen -------------------------------------------------------------

# Der migrate-Container teilt sich den Netzwerk-Namespace mit Postgres. Dadurch trifft
# das localhost:5432 aus DATABASE_URL genau die Datenbank, und der bereits korrekt
# prozent-kodierte Wert aus backend\.env lässt sich unverändert weiterreichen - der
# migrate-Service in docker-compose.yml baut die URL dagegen selbst zusammen und
# zerbricht an Sonderzeichen im Passwort (Issue #71).
Write-Log 'Migrationen einspielen ...'
docker run --rm `
    --network "container:$pgContainer" `
    -v "${root}\backend\migrations:/migrations:ro" `
    migrate/migrate:v4.19.1 `
    -path=/migrations `
    "-database=$($backendEnv['DATABASE_URL'])" `
    up
if ($LASTEXITCODE -ne 0) { Stop-WithError 'Migrationen sind fehlgeschlagen.' }

# --- Frontend-Abhängigkeiten -------------------------------------------------

# Ein Checkout, der Dependencies hinzufügt, lässt node_modules veralten. Vite meldet das
# als "Failed to resolve import", was nach einem Codefehler aussieht statt nach einem
# Setup-Problem.
$modules = Join-Path $root 'frontend\node_modules'
$manifest = Join-Path $root 'frontend\package.json'
if (-not (Test-Path $modules) -or
        (Get-Item $manifest).LastWriteTime -gt (Get-Item $modules).LastWriteTime) {
    Write-Log 'Frontend-Abhängigkeiten sind veraltet, npm ci läuft ...'
    Push-Location 'frontend'
    npm ci
    Pop-Location
    if ($LASTEXITCODE -ne 0) { Stop-WithError 'npm ci ist fehlgeschlagen.' }
}

# --- Start -------------------------------------------------------------------

# mprocs gibt jedem Prozess einen eigenen scrollbaren Bereich und ein eigenes
# Pseudo-Terminal. Dadurch bleiben die Farben von Gin und Vite erhalten, die über die
# umgeleiteten Streams im Fallback unten verloren gehen. Optional, damit das Skript ohne
# zusätzliche Installation funktioniert.
#
# Landing Page und Dev-Proxy sind in mprocs.yaml als eigene Prozesse hinterlegt
# (landing, proxy) -- mprocs startet sie automatisch mit, kein Zutun hier nötig.
if (Get-Command mprocs -ErrorAction SilentlyContinue) {
    Write-Log 'Starte in getrennten Bereichen (mprocs).'
    mprocs --config (Join-Path $root 'mprocs.yaml')
    exit $LASTEXITCODE
}

Write-Log 'Tipp: mit mprocs bekommt jeder Prozess einen eigenen Bereich und seine Farben zurück.'
Write-Log "      Installieren z.B. mit 'pnpm add -g mprocs' oder 'npm i -g mprocs'."

$processes = @()

function Start-Prefixed($tag, $color, $directory, $command, $commandArgs) {
    $info = New-Object System.Diagnostics.ProcessStartInfo
    $info.FileName = $command
    $info.Arguments = $commandArgs
    $info.WorkingDirectory = Join-Path $root $directory
    $info.RedirectStandardOutput = $true
    $info.RedirectStandardError = $true
    $info.UseShellExecute = $false

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $info

    $handler = {
        if ($EventArgs.Data) {
            Write-Host "[$($Event.MessageData.Tag)] " -ForegroundColor $Event.MessageData.Color -NoNewline
            Write-Host $EventArgs.Data
        }
    }
    $messageData = [PSCustomObject]@{ Tag = $tag; Color = $color }
    Register-ObjectEvent -InputObject $process -EventName OutputDataReceived -Action $handler -MessageData $messageData | Out-Null
    Register-ObjectEvent -InputObject $process -EventName ErrorDataReceived -Action $handler -MessageData $messageData | Out-Null

    $process.Start() | Out-Null
    $process.BeginOutputReadLine()
    $process.BeginErrorReadLine()
    return $process
}

try {
    $processes += Start-Prefixed 'backend' 'Cyan' 'backend' 'go' 'run .'
    # `npm` löst auf diesem System zu npm.ps1 auf (Get-Command npm bestätigt
    # das), nicht zu npm.cmd. cmd.exe kennt aber nur .cmd/.bat/.exe und würde
    # bei der PATH-Suche eine andere, hier nicht funktionierende npm.cmd
    # treffen ("Cannot find module ...npm-prefix.js" mit falsch
    # zusammengesetztem Pfad). powershell.exe -Command nutzt exakt denselben
    # Auflösungsmechanismus wie ein manuell getipptes `npm run dev` und trifft
    # daher zuverlässig npm.ps1.
    $processes += Start-Prefixed 'frontend' 'Green' 'frontend' 'powershell.exe' '-NoProfile -Command "npm run dev"'
    # Landing Page läuft im Repo-Root (nicht landing/), sonst kein Zugriff auf
    # den Geschwisterordner shared/fonts/. cwd '.' entspricht $root selbst.
    $processes += Start-Prefixed 'landing' 'Magenta' '.' 'powershell.exe' "-NoProfile -Command `"npx serve . -l $landingPort`""
    $processes += Start-Prefixed 'proxy' 'Yellow' 'dev-proxy' 'node' 'proxy.js'

    Write-Log "Backend auf :$backendPort, Frontend auf :$frontendPort, Landing auf :$landingPort, Proxy auf :$proxyPort."
    Write-Log "Alles zusammen erreichbar unter http://localhost:$proxyPort. Beenden mit Strg-C."

    # Sobald einer der vier weg ist, sollen auch die anderen gehen - ein halber Stack
    # hilft niemandem.
    while (-not ($processes | Where-Object { $_.HasExited })) {
        Start-Sleep -Seconds 1
    }
    Write-Log 'Ein Prozess hat sich beendet - die anderen werden gestoppt.'
} finally {
    # Stop-Process beendet nur den Prozess selbst. `go run` startet die kompilierte
    # Binärdatei als Enkelkind, das sonst überlebt und Port 8080 belegt hält; taskkill
    # mit /T nimmt den ganzen Baum mit.
    foreach ($process in $processes) {
        if ($process -and -not $process.HasExited) {
            taskkill /PID $process.Id /T /F 2>$null | Out-Null
        }
    }
}