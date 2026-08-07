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

Die Prüfungen laufen gegen den **Arbeitsbaum**, nicht gegen den Commit — ein halbfertiger
Umbau blockiert den Push also auch dann, wenn der Commit selbst sauber ist.

Zwischenstände trotzdem pushen: Betreff des obersten Commits mit `wip` beginnen lassen.
Nur diesen einen Betreff liest der Hook, und er überspringt dann alle Prüfungen — auch
für die sauberen Commits darunter.
Vor dem PR gehören diese Commits zusammengefasst oder umbenannt. Harter Notausgang bleibt
`git push --no-verify`.
