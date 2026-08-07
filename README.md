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

Der Hook prüft den **Arbeitsbaum**, nicht den Commit — ein halbfertiger Umbau blockiert
den Push also auch dann, wenn der Commit selbst sauber ist.

Zwischenstände trotzdem pushen: Betreff mit `wip` beginnen lassen, dann überspringt der
Hook alle Prüfungen. Gilt für den ganzen Push, sobald der oberste Commit ein `wip` ist.
Vor dem PR gehören diese Commits zusammengefasst oder umbenannt. Harter Notausgang bleibt
`git push --no-verify`.
