#!/usr/bin/env bash
#
# Misst Durchsatz, Latenz und Spitzen-RSS von POST /api/auth/register über mehrere
# Clientzahlen. Belegt die Maschine für die Dauer des Laufs vollständig -- siehe die
# Warnungen unten, sie sind keine Formsache.
#
# Written for bash 3.2, the version macOS still ships.
#
# Beispiele:
#   set -a; . backend/.env; set +a            # DSN in die Shell, einmal pro Terminal
#   scripts/loadtest.sh                       # Default-Grenze (GOMAXPROCS)
#   AUTH_HASH_CONCURRENCY=4 scripts/loadtest.sh
#   AUTH_HASH_CONCURRENCY=1000 scripts/loadtest.sh   # praktisch unbegrenzt, Vorher-Wert
#   LEVELS="8 32" DURATION=5s scripts/loadtest.sh

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

LEVELS="${LEVELS:-4 8 16 32 64}"
DURATION="${DURATION:-12s}"
PORT="${PORT:-8080}"
WORK="$(mktemp -d)"

GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'

log()  { printf "${GREEN}[load]${RESET} %s\n" "$1"; }
warn() { printf "${YELLOW}[load]${RESET} %s\n" "$1"; }
fail() { printf "${RED}[load]${RESET} %s\n" "$1" >&2; exit 1; }

# Im Postgres-Container zeigt das localhost aus der DSN auf genau diese Datenbank,
# deshalb laesst sich DATABASE_URL unveraendert weiterreichen.
cleanup() {
    if [ -n "${sampler_pid:-}" ]; then kill "$sampler_pid" 2>/dev/null || true; fi
    if [ -n "${server_pid:-}" ]; then kill "$server_pid" 2>/dev/null || true; fi

    if [ -n "${users_created:-}" ] && [ -n "${DATABASE_URL:-}" ]; then
        log "raeume Testnutzer ab"
        if docker compose exec -T postgres psql "$DATABASE_URL" \
            -c "DELETE FROM users WHERE email LIKE 'loadtest-%@example.com';" >/dev/null 2>&1; then
            log "Testnutzer geloescht"
        else
            warn "Aufraeumen fehlgeschlagen. Von Hand:"
            warn "  DELETE FROM users WHERE email LIKE 'loadtest-%@example.com';"
        fi
    fi

    rm -rf "$WORK"
}
trap cleanup EXIT

cat <<'WARNUNG'

  Dieser Lauf fährt Argon2id über Minuten auf allen Kernen aus.

  - Die Maschine wird spürbar heiß, Lüfter drehen hoch. Auf Laptops drosselt
    die CPU dabei irgendwann selbst, und ab da misst du die Drosselung statt
    des Servers. Für belastbare Zahlen: Netzteil dran, Deckel offen, und
    zwischen zwei Läufen abkühlen lassen.
  - Andere Programme auf derselben Maschine verfälschen die Messung und werden
    ihrerseits langsam. Browser und Build-Prozesse vorher schließen.
  - Der Lauf legt zehntausende Nutzer an und löscht sie am Ende wieder.
    Nur gegen eine Wegwerf-Datenbank laufen lassen.
  - Absolute Zahlen gelten nur für diese Maschine. Vergleichbar sind
    ausschließlich zwei Läufe auf derselben Hardware.

WARNUNG

printf "  Weiter? [j/N] "
read -r answer
case "$answer" in
    j|J|y|Y) ;;
    *) fail "abgebrochen" ;;
esac

# Wie bei `go test`: die DSN kommt aus der Shell, nicht aus einer hier gesourcten Datei.
# `. backend/.env` würde den Inhalt als Bash ausführen, siehe dev.sh.
[ -n "${DATABASE_URL:-}" ] || fail "DATABASE_URL fehlt. Vorher: set -a; . backend/.env; set +a"
export COOKIE_SECURE=false
export GIN_MODE=release

log "baue Server und Lastgenerator"
(cd backend && go build -o "$WORK/server" . && go build -o "$WORK/loader" ./cmd/loadtest)

tag="$(date +%H%M%S)"
users_created=yes

# Der Server wird pro Messpunkt neu gestartet: RSS ist eine Hochwassermarke, ohne
# Neustart trüge jeder Messpunkt die Spitze des vorherigen mit sich herum.
for level in $LEVELS; do
    "$WORK/server" >"$WORK/server.log" 2>&1 &
    server_pid=$!

    ready=""
    for _ in $(seq 1 50); do
        code=$(curl -s -o /dev/null -w '%{http_code}' -m 2 -X POST \
            -H 'content-type: application/json' -d '{}' \
            "http://127.0.0.1:$PORT/api/auth/register" || true)
        if [ "$code" != "000" ]; then ready=yes; break; fi
        sleep 0.2
    done
    [ -n "$ready" ] || { cat "$WORK/server.log" >&2; fail "Server auf Port $PORT nicht erreichbar"; }

    echo 0 >"$WORK/peak"
    ( peak=0
      while kill -0 "$server_pid" 2>/dev/null; do
          rss=$(ps -o rss= -p "$server_pid" 2>/dev/null | tr -d ' ')
          if [ -n "$rss" ] && [ "$rss" -gt "$peak" ]; then
              peak=$rss
              echo "$peak" >"$WORK/peak"
          fi
          sleep 0.2
      done ) &
    sampler_pid=$!

    result=$("$WORK/loader" -c "$level" -d "$DURATION" -tag "$tag-$level" \
        -url "http://127.0.0.1:$PORT/api/auth/register")

    kill "$sampler_pid" 2>/dev/null || true
    wait "$sampler_pid" 2>/dev/null || true
    sampler_pid=""
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
    server_pid=""

    printf '%s rss=%s MiB\n' "$result" "$(( $(cat "$WORK/peak") / 1024 ))"
done
