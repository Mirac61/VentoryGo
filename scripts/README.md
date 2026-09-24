# Dev-Skripte

Ein Befehl fährt den kompletten Stack hoch. Welches Skript, hängt vom Betriebssystem ab:

```bash
./scripts/dev.sh      # Linux, macOS
.\scripts\dev.ps1     # Windows (PowerShell 7 oder neuer)
```

Beide tun dasselbe in derselben Reihenfolge. Wer eines ändert, ändert das andere mit —
sonst driften die Plattformen auseinander und niemand merkt es, bis es jemanden trifft.
Dazu gehört auch, `backend/.env` in beiden als Schlüssel-Wert-Datei zu lesen statt sie
als Shell-Code auszuführen: Sonst ist ein unquotierter Wert mit Leerzeichen unter Bash
ein Kommandoaufruf und unter PowerShell einfach ein Wert.

Strg-C beendet Backend und Frontend. Postgres bleibt absichtlich stehen: Es startet
langsamer als der Rest, und beim nächsten Lauf ist es dann sofort bereit.

## Was passiert, in dieser Reihenfolge

1. **Prüfen, ob die Voraussetzungen da sind** — `docker`, `go`, `npm` und beide
   `.env`-Dateien. Fehlt etwas, steht der passende Befehl in der Fehlermeldung.
2. **Port 8080 prüfen.** Ist er belegt, bricht das Skript sofort ab statt später mit
   einem `address already in use` aus dem Backend.
3. **Postgres starten** und auf den Healthcheck warten. Ein Container, der läuft, nimmt
   noch keine Verbindungen an — deshalb wird gewartet und nicht geraten.
4. **Migrationen einspielen.** Ohne Schema antwortet die API mit 500 statt zu scheitern,
   was beim Suchen in die falsche Richtung führt.
5. **`npm ci`, falls `node_modules` älter ist als `package.json`.** Ein Branch, der
   Dependencies hinzufügt, lässt die Installation veralten; Vite meldet das als
   `Failed to resolve import`, was nach einem Codefehler aussieht statt nach einem
   Setup-Problem.
6. **Backend und Frontend starten.**

## Getrennte Bereiche statt einer gemeinsamen Ausgabe

Ist [mprocs](https://github.com/pvolok/mprocs) installiert, übergeben die Skripte daran
und `mprocs.yaml` legt drei Bereiche an: `backend`, `frontend` und `docker`. Jeder ist
einzeln scrollbar, und `docker` startet auf Zuruf, weil `docker stats` sich sekündlich
neu zeichnet.

Der eigentliche Gewinn sind die Farben. Gin und Vite prüfen, ob sie in ein Terminal
schreiben, und schalten die Einfärbung sonst ab — die gemeinsame Ausgabe des Fallbacks
läuft durch eine Pipe und verliert sie deshalb. mprocs gibt jedem Prozess ein eigenes
Pseudo-Terminal, damit sind sie zurück.

Installation, falls gewünscht:

```bash
pnpm add -g mprocs    # landet im Home
npm i -g mprocs       # braucht je nach Node-Installation Rechte auf /usr/local
```

**Optional.** Ohne mprocs läuft alles wie bisher, mit nach Herkunft beschrifteter
Ausgabe in einem Terminal.

## Wenn etwas schiefgeht

| Meldung oder Symptom | Ursache |
|---|---|
| `Port 8080 ist belegt` | Ein Backend aus einem früheren Lauf lebt noch. `pkill -f "go run"`, unter Windows `taskkill /IM backend.exe /T /F`. |
| `.env fehlt` | Die Datei ist nicht im Repo, weil sie Zugangsdaten enthält. Der `cp`-Befehl steht in der Meldung. |
| `DATABASE_URL fehlt oder ist leer` | `backend/.env` existiert, aber unvollständig — meist eine aus `.env.example` kopierte Datei, in der die Werte noch nicht gesetzt wurden. |
| `Postgres wurde nicht healthy` | `docker compose logs postgres` sagt warum. Meist ein Passwort, das nicht zum bestehenden Volume passt. |
| `Failed to resolve import` in Vite | `node_modules` ist älter als der Branch. Das Skript fängt das ab; von Hand: `cd frontend && npm ci`. |
| `failed to connect to database: /tmp/.s.PGSQL.5432` | `DATABASE_URL` war nicht gesetzt. `go run .` liest **keine** `.env`, `main.go` ruft nur `os.Getenv`. Das Skript setzt sie; von Hand braucht es `set -a; . ./backend/.env; set +a`. |
| Es erscheint die React-Startseite statt der Anmeldung | Falscher Branch. Das Skript liefert aus, was im Arbeitsbaum liegt. |
| `invalid character "}" in host name` | Nur auf dem `--profile docker`-Pfad: Compose setzt das Passwort ungekodiert in eine URL ein. Siehe Issue #71 — die Skripte umgehen das. |

## Was die Skripte bewusst nicht tun

- **Postgres beim Beenden stoppen.** Siehe oben.
- **Die Test-Datenbank anlegen.** Die läuft nicht im Alltag mit, die Einrichtung steht
  in [CONTRIBUTING.md](../CONTRIBUTING.md#tests).
- **Den `--profile docker`-Pfad ersetzen.** Der bleibt für „einmal komplett in Containern
  hochfahren". Im Alltag läuft das Backend auf dem Host, damit eine Codeänderung keinen
  Image-Rebuild kostet und der Debugger sich nativ anhängen kann.
