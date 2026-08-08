#!/usr/bin/env bash
#
# Starts the full dev stack in the right order: Postgres, migrations, backend, frontend.
# Windows users: use dev.ps1 instead.
#
# Written for bash 3.2, the version macOS still ships, and without GNU-only flags --
# `stat -c` and `readlink -f` do not exist there.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BACKEND_PORT=8080
FRONTEND_PORT=5173

CYAN='\033[36m'
GREEN='\033[32m'
RED='\033[31m'
RESET='\033[0m'

log()  { printf "${GREEN}[dev]${RESET} %s\n" "$1"; }
fail() { printf "${RED}[dev]${RESET} %s\n" "$1" >&2; exit 1; }

# --- Cleanup -----------------------------------------------------------------

# Ohne das überlebt ein Ctrl-C in den Kindprozessen: `go run` hält seine kompilierte
# Binärdatei auf Port 8080 fest, und der nächste Start scheitert mit "address already
# in use". Über Prozessgruppen geht das nicht zuverlässig -- `set -m` legt in einem
# Skript ohne Terminal keine eigenen Gruppen an -- deshalb wird der Baum anhand der
# Elternbeziehung abgelaufen. Das Format von `ps` verstehen BSD und GNU gleichermaßen.
kill_tree() {
    for child in $(ps -A -o pid=,ppid= | awk -v parent="$1" '$2 == parent { print $1 }'); do
        kill_tree "$child"
    done
    kill -TERM "$1" 2>/dev/null || true
}

cleaned=""
child_pids=""
cleanup() {
    [ -n "$cleaned" ] && return
    cleaned=1
    trap - EXIT INT TERM
    [ -n "$child_pids" ] && log "Stoppe Backend und Frontend …"
    for pid in $child_pids; do
        kill_tree "$pid"
    done
}
trap cleanup EXIT INT TERM

# --- Preflight ---------------------------------------------------------------

require_command() {
    command -v "$1" >/dev/null 2>&1 || fail "$1 nicht gefunden. $2"
}

require_file() {
    [ -f "$1" ] || fail "$1 fehlt. Anlegen mit: $2"
}

port_in_use() {
    (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null && exec 3>&- && return 0
    return 1
}

require_command docker "Docker Desktop bzw. den Docker-Daemon starten."
require_command go "Go installieren: https://go.dev/dl/"
require_command npm "Node installieren: https://nodejs.org/"

require_file ".env" "cp .env.example .env"
require_file "backend/.env" "cp backend/.env.example backend/.env"

if port_in_use "$BACKEND_PORT"; then
    fail "Port $BACKEND_PORT ist belegt. Läuft noch ein altes Backend?"
fi
if port_in_use "$FRONTEND_PORT"; then
    log "Port $FRONTEND_PORT ist belegt — Vite weicht auf einen anderen aus."
fi

# `go run .` liest keine .env: main.go ruft nur os.Getenv. Ohne das hier fällt pgx auf
# die libpq-Defaults zurück und meldet einen Unix-Socket-Fehler, der nach einem
# kaputten Postgres aussieht.
set -a
# shellcheck disable=SC1091
. ./backend/.env
set +a

# --- Postgres ----------------------------------------------------------------

log "Postgres starten …"
docker compose up -d postgres

PG_CONTAINER="$(docker compose ps -q postgres)"
[ -n "$PG_CONTAINER" ] || fail "Postgres-Container nicht gefunden."

log "Warte, bis Postgres Verbindungen annimmt …"
waited=0
while [ "$(docker inspect --format '{{.State.Health.Status}}' "$PG_CONTAINER")" != "healthy" ]; do
    sleep 1
    waited=$((waited + 1))
    [ "$waited" -lt 60 ] || fail "Postgres wurde nicht healthy. Siehe: docker compose logs postgres"
done

# --- Migrationen -------------------------------------------------------------

# Der migrate-Container teilt sich den Netzwerk-Namespace mit Postgres. Dadurch trifft
# das `localhost:5432` aus DATABASE_URL genau die Datenbank, und der bereits korrekt
# prozent-kodierte Wert aus backend/.env lässt sich unverändert weiterreichen -- der
# migrate-Service in docker-compose.yml baut die URL dagegen selbst zusammen und
# zerbricht an Sonderzeichen im Passwort (Issue #71).
log "Migrationen einspielen …"
docker run --rm \
    --network "container:$PG_CONTAINER" \
    -v "$ROOT/backend/migrations:/migrations:ro" \
    migrate/migrate:v4.19.1 \
    -path=/migrations \
    -database="$DATABASE_URL" \
    up

# --- Frontend-Abhängigkeiten -------------------------------------------------

# Ein Checkout, der Dependencies hinzufügt, lässt node_modules veralten. Vite meldet
# das als "Failed to resolve import", was nach einem Codefehler aussieht statt nach
# einem Setup-Problem.
if [ ! -d frontend/node_modules ] || [ frontend/package.json -nt frontend/node_modules ]; then
    log "Frontend-Abhängigkeiten sind veraltet, npm ci läuft …"
    (cd frontend && npm ci)
fi

# --- Start -------------------------------------------------------------------

# mprocs gibt jedem Prozess einen eigenen scrollbaren Bereich und ein eigenes
# Pseudo-Terminal. Dadurch bleiben die Farben von Gin und Vite erhalten, die durch die
# Pipe des Fallbacks unten verloren gehen. Optional, damit das Skript ohne zusätzliche
# Installation funktioniert.
if command -v mprocs >/dev/null 2>&1; then
    log "Starte in getrennten Bereichen (mprocs)."
    exec mprocs --config "$ROOT/mprocs.yaml"
fi

log "Tipp: mit mprocs bekommt jeder Prozess einen eigenen Bereich und seine Farben zurück."
log "      Installieren z.B. mit 'pnpm add -g mprocs' — 'npm i -g' braucht je nach"
log "      Node-Installation Schreibrechte auf /usr/local."

run_prefixed() {
    tag="$1"
    color="$2"
    directory="$3"
    shift 3
    (
        cd "$directory"
        "$@" 2>&1 | while IFS= read -r line; do
            printf "${color}[%s]${RESET} %s\n" "$tag" "$line"
        done
    ) &
}

run_prefixed backend "$CYAN" backend go run .
backend_pid=$!

run_prefixed frontend "$GREEN" frontend npm run dev
frontend_pid=$!

child_pids="$backend_pid $frontend_pid"

log "Backend auf :$BACKEND_PORT, Frontend auf :$FRONTEND_PORT. Beenden mit Strg-C."

# bash 3.2 kennt kein `wait -n`, deshalb pollen statt warten: sobald einer der beiden
# Prozesse weg ist, soll auch der andere gehen -- ein halber Stack hilft niemandem.
while kill -0 "$backend_pid" 2>/dev/null && kill -0 "$frontend_pid" 2>/dev/null; do
    sleep 1
done

log "Ein Prozess hat sich beendet — der andere wird gestoppt."
