# Architecture Decision Records

Entscheidungen, die sich durch Folgetickets ziehen, stehen hier — mit dem Kontext, in dem
sie gefallen sind. Der Zweck ist nicht Dokumentation des Ist-Zustands, sondern die Antwort
auf die Frage "warum eigentlich so?", wenn sie in einem halben Jahr jemand stellt.

## Konvention

- Dateiname `NNNN-kurzer-titel.md`, fortlaufend ab `0001`
- Format nach Michael Nygard, aber BLUF: Entscheidung zuerst (was + warum), dann kurz
  verworfene Alternativen, dann Kontext/Konsequenzen/Offene Punkte — wer's eilig hat,
  liest nur den ersten Absatz
- Status: `Vorgeschlagen`, `Angenommen`, `Ersetzt durch NNNN`

Sobald eine ADR in `main` steht, wird sie nicht mehr geändert. Ändert sich die Entscheidung,
entsteht eine neue ADR, und die alte bekommt den Status `Ersetzt durch NNNN`. Sonst geht
genau das verloren, wofür die Sammlung da ist: dass eine Entscheidung damals aus damaligen
Gründen richtig war.

Solange sie noch in einem offenen PR steckt, wird sie normal überarbeitet — dort ist sie
Entwurf und niemand hat sich darauf verlassen.

Kleinkram gehört nicht hierher. Eine ADR lohnt sich, wenn eine Umkehr teuer wäre oder wenn
die naheliegende Alternative bewusst verworfen wurde.

## Übersicht

| Nr. | Titel | Status |
|---|---|---|
| [0001](0001-authentifizierung.md) | Authentifizierung | Angenommen |
| [0002](0002-pdf-bibliothek.md) | PDF-Bibliothek | Angenommen |



