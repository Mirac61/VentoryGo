# Invoice

Rechnungsverwaltung: Go/Gin-Backend + React-Frontend.

| Verzeichnis | Inhalt |
|---|---|
| [`backend/`](backend/README.md) | REST-API, Go/Gin, Postgres |
| [`frontend/`](frontend/README.md) | React-Client |
| [`scripts/`](scripts/README.md) | Dev-Stack mit einem Befehl starten |

## Einmalig nach dem Clone

```bash
git config core.hooksPath .githooks
```

Aktiviert die versionierten Git-Hooks (`.githooks/pre-push`: gofmt, vet, Tests).
Git setzt das absichtlich nicht automatisch — sonst könnte jedes geclonte Repo
ungefragt Code ausführen. Ohne diesen Befehl läuft der Hook stillschweigend nie.

Die Prüfungen laufen gegen den **Arbeitsbaum**, nicht gegen den Commit — ein halbfertiger
Umbau blockiert den Push also auch dann, wenn der Commit selbst sauber ist.

Zwischenstände trotzdem pushen: Betreff des obersten Commits mit `wip` beginnen lassen.
Nur diesen einen Betreff liest der Hook, und er überspringt dann alle Prüfungen — auch
für die sauberen Commits darunter.
Vor dem PR gehören diese Commits zusammengefasst oder umbenannt. Harter Notausgang bleibt
`git push --no-verify`.

## Loslegen

Beide `.env`-Dateien anlegen — wie, steht in [CONTRIBUTING.md](CONTRIBUTING.md#setup).
Danach fährt ein Befehl Postgres, Migrationen, Backend und Frontend hoch:

```bash
./scripts/dev.sh      # Linux, macOS
.\scripts\dev.ps1     # Windows
```

Was dabei passiert und was die häufigen Fehlermeldungen bedeuten, steht in
[`scripts/README.md`](scripts/README.md).
