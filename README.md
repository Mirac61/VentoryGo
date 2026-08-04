# Invoice

Rechnungsverwaltung: Go/Gin-Backend + React-Frontend.

| Verzeichnis | Inhalt |
|---|---|
| [`backend/`](backend/README.md) | REST-API, Go/Gin, Postgres |
| [`frontend/`](frontend/README.md) | React-Client |

## Einmalig nach dem Clone

```bash
git config core.hooksPath .githooks
```

Aktiviert die versionierten Git-Hooks (`.githooks/pre-push`: gofmt, vet, Tests).
Git setzt das absichtlich nicht automatisch — sonst könnte jedes geclonte Repo
ungefragt Code ausführen. Ohne diesen Befehl läuft der Hook stillschweigend nie.

Notausgang für WIP-Branches: `git push --no-verify`.
