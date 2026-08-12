# 2. PDF-Bibliothek

Status: Angenommen
Datum: 2026-08-12
Ticket: [#23](https://github.com/Mirac61/VentoryGo/issues/23), Grundlage für
[#24](https://github.com/Mirac61/VentoryGo/issues/24)–[#28](https://github.com/Mirac61/VentoryGo/issues/28)

## Entscheidung

`github.com/johnfercher/maroto/v2`, Version 2.4.0, MIT-Lizenz. Die Entscheidung fiel
ganz leicht, da Maroto eine einfachere Erstellung von PDFs ermöglicht. Es arbeitet mit
einem Grid-Raster statt freier Koordinaten, das ließ sich im Prototyp aber ohne
Einschränkung bei Farbe, Schrift und Logo anpassen. Maroto rendert dabei selbst über
`phpdave11/gofpdf`.

Diese Entscheidung kam erst nach einem Wegwerf-Prototypen (60 Positionen, 3 Seiten,
farbiger Tabellenkopf, eingebettete TTF); Renderbeispiele daraus lassen sich bei Bedarf
als Anhang an #23 nachreichen. Er zeigte:

- **Tabellenkopf über Seiten:** `RegisterHeader` wiederholt die Kopfzeile automatisch.
- **Umlaute:** werden korrekt angezeigt.
- **Farbe, Schrift, Logo austauschbar:** `props.Cell{BackgroundColor}`,
  `props.Text{Color}`, `config.WithCustomFonts`, `image.NewFromBytesCol`.
- **Kein externes Binary nötig:** `CGO_ENABLED=0 GOOS=linux go build`.
- **Zeit/Speicher:** ~8 ms und 18 MB Allokationen je dreiseitiger Rechnung, mit einer
  186-KB-TTF und einmal geladener Schrift.

PDF/A-3 war kein Kriterium bei dieser Wahl: Keine der Optionen erzeugt es von allein,
das bleibt so oder so ein Nachbearbeitungsschritt
([#66](https://github.com/Mirac61/VentoryGo/issues/66),
[#68](https://github.com/Mirac61/VentoryGo/issues/68)).

## Verworfene Alternativen

**HTML-Template → Headless Chrome oder wkhtmltopdf.** Bringt ein zweites
Laufzeit-Artefakt ins Docker-Image, statt nur das eine statische Go-Binary zu bauen.
Und Seitenumbruch mit wiederholtem Tabellenkopf ist in Druck-CSS notorisch
unzuverlässig — genau das, worum es bei der Positionstabelle in
[#25](https://github.com/Mirac61/VentoryGo/issues/25) geht.

**gofpdf direkt, mit rohen Koordinaten.** Eigentlich kein echter Gegenkandidat:
Maroto rendert selbst über gofpdf, das wäre also dieselbe Engine ohne das Grid
darüber. Ohne Maroto baut man Spaltenbreiten, Umbruch und Kopfwiederholung von Hand —
genau das, was in [#25](https://github.com/Mirac61/VentoryGo/issues/25) sowieso
ansteht. Dass sich ein fremdes Grid-Modell schlecht parametrisieren lässt, hat sich im
Prototyp nicht bestätigt: Farbe, Schrift und Logo sind einfach Werte, die man
reinreicht, keine feste Struktur.

## Konsequenzen

**Schriften werden einmal beim Start geladen**, nicht pro Request. Der
Speicherbedarf pro Rechnung hängt an der Schriftdatei — mit einer 23-MB-Unicode-Schrift
lag der Test bei 130 MB statt 18 MB. Die ausgelieferte Schrift muss also klein bleiben.

**Das Layout steht in Go-Code, nicht in einer Vorlagendatei.** Farbe, Schrift, Logo
und freie Texte sind davon ausgenommen, die kommen aus dem Theme
([#75](https://github.com/Mirac61/VentoryGo/issues/75)). Am Raster selbst ist jede
Änderung ein Deployment.

**Zwölf Spalten, Zeilen von oben nach unten.** Freie Positionierung oder
überlappende Elemente sind unbequem, und einen offiziellen Ausstieg auf Koordinaten
gibt es nicht — der gofpdf-Provider liegt in `internal/`.

**Nur ein wiederholter Kopfbereich.** Was in `RegisterHeader` steht, erscheint auf
jeder Seite. Ein Briefkopf, der nur auf Seite 1 gehört, und ein Tabellenkopf, der sich
wiederholen soll, können nicht beide dort stehen. Zu klären in
[#24](https://github.com/Mirac61/VentoryGo/issues/24).

**Rund 30 Module kommen mit**, darunter pdfcpu (Apache-2.0) und ein
Barcode-Renderer. Keines davon braucht CGO, alle Lizenzen sind permissiv.

**Der Renderer bekommt ein eigenes Eingabe-Struct**, kein DB-Model und kein
`*gin.Context`. Maroto-Typen verlassen das Renderer-Package nicht. Ein Wechsel der
Bibliothek trifft damit nur dieses eine Package.

## Offene Punkte

- **Der Prototyp lief auf dem Entwicklungsrechner, nicht im Image.**
    - `CGO_ENABLED=0 GOOS=linux go build` prüft nur den Binary-Build, nicht das
      Laufzeit-Image — ob Fonts und Assets darin ankommen, ist ungetestet
    - Es fehlt ein CI-Test, der das Image baut, darin ein PDF erzeugt und validiert
    - Zeiten/Speicherwerte stammen vom Entwicklungsrechner, nicht aus dem Image

- **Schriftart / Font ist nicht festgelegt worden**:
    - Es muss lizenzrechtlich erlaubt sein
    - Neue Schriftart wirkt erst ab dem nächsten generierten PDF, nicht rückwirkend
    - bereits erzeugte Rechnungen behalten ihre alte Schrift
    - Kollege erwartete zunächst rückwirkende Änderung — bewusst nicht so umgesetzt
    - passt zur Vorgabe aus #75: alte Rechnungen ändern sich bei Preset-Wechsel nicht
    - latein muss abgedeckt werden

- **Seitenformat ist noch nicht ans Theme angebunden.** Maroto unterstützt A3, A4,
  A5, Letter und Legal fertig über `WithPageSize(...)` — das ist keine Einschränkung
  der Bibliothek, nur noch nicht mit dem Firmenprofil aus #75 verdrahtet

