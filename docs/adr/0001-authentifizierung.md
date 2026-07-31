# 1. Authentifizierung

Status: Angenommen
Datum: 2026-07-31
Ticket: [#44](https://github.com/Mirac61/VentoryGo/issues/44), Teil von [#36](https://github.com/Mirac61/VentoryGo/issues/36)

## Kontext

Bisher gibt es keine Anmeldung. Mit dem Firmenprofil aus #36 kommt sie, und mit ihr die
Frage, woran der Server einen Request einem Nutzer zuordnet. Die Entscheidung fällt vor den
Tabellen und Endpoints, weil sie sich sonst durch jedes Folgeticket zieht und dort jedes
Mal neu diskutiert wird.

Die Ausgangslage bestimmt, welche Optionen überhaupt in Frage kommen:

- Ein Go-Backend, ein Browser-Frontend, beide unter derselben Origin
- Keine Service-zu-Service-Aufrufe, kein mobiler Client geplant
- Postgres ist gesetzt und wird auf jedem Request ohnehin angefasst
- Rechnungs- und Firmendaten, also personenbezogen und geschäftlich
- Anmeldung per E-Mail und Passwort

Der letzte Punkt aus der Liste ist der wichtigste für alles Weitere: eine Sitzung muss
beendet werden können. Nicht nur "der Browser vergisst das Cookie", sondern "ab jetzt
kommt mit diesem Token niemand mehr rein". Das brauchen Logout, der Passwortwechsel und
der Fall, dass jemand seinen Laptop im Zug liegen lässt.

## Betrachtete Optionen

### Token im Client (JWT oder signiertes Cookie)

Der Server merkt sich nichts. Im Token stehen `user_id` und Ablaufzeit, dazu eine Signatur,
die nur der Server prüfen kann. Kein Speicher, kein Lookup, beliebig viele Instanzen ohne
gemeinsamen Zustand.

Der übliche Einwand — Token im `localStorage`, damit ist jede XSS-Lücke ein Vollzugriff auf
das Konto — trifft nur die JWT-Variante. Legt man dasselbe signierte Token in ein
`HttpOnly`-Cookie, ist es gegen Auslesen per JavaScript genauso geschützt wie eine
Session-ID. Diese Begründung allein reicht also nicht.

Was bleibt, ist der Widerruf. Ein gültig signiertes Token kann der Server nicht
zurückholen:

- Beim Logout wird nur das Cookie im Browser überschrieben. Wer den Token-String vorher
  kopiert hat, bleibt bis zum Ablauf angemeldet.
- "Auf allen Geräten abmelden" nach einem Passwortwechsel ist nicht umsetzbar.
- Ändert sich die Firmenzugehörigkeit oder die Rolle, gilt der alte Stand aus dem Token
  weiter.

Der Standardausweg dagegen ist kurze Laufzeit plus Refresh-Token mit Rotation und einer
Sperrliste für widerrufene Tokens. Damit ist der Server wieder zustandsbehaftet, nur mit
zwei Token-Typen und einer Rotationslogik obendrauf. Das ist mehr Mechanik als die Tabelle,
die es ersetzen sollte.

### Server-seitige Session, Store in Redis

Löst dasselbe Problem wie eine Tabelle in Postgres und erlaubt genauso den Widerruf. Zwei
echte Vorteile: die Ablaufzeit ist eingebaut (`EX 604800`), also entfällt das Aufräumen,
und das Verlängern einer Session ist ein billiger Schreibvorgang im Speicher statt eines
`UPDATE` in der Datenbank.

Dagegen steht ein zweiter Dienst in `docker-compose.yml`, im Code, in den Tests und im
Betrieb. Dazu eine Falle: ohne konfigurierte Persistenz verliert Redis beim Neustart alle
Schlüssel, jedes Deployment wäre also ein Logout für alle. Der Vorteil beim Verlängern
verschwindet ohnehin durch die 15-Minuten-Regel weiter unten.

### Fremde Lösung (Keycloak, Auth-SaaS)

Nimmt einem Passwort-Hashing, Passwort-vergessen-Flows und später OAuth ab. Der Preis ist
ein weiterer Dienst mit eigenem Datenmodell, eigenem Update-Zyklus und eigener
Fehlersuche — bei einer SaaS zusätzlich eine externe Abhängigkeit für den Login.

Für E-Mail und Passwort gegen einen einzigen Client steht dem kein Gegenwert gegenüber.
Der Teil, den Keycloak hier abnehmen würde, sind rund hundert Zeilen Go.

## Entscheidung

### Server-seitige Sessions in einer Postgres-Tabelle

Der Store liegt in Postgres, weil die Datenbank auf jedem Request ohnehin angefasst wird.
Der Session-Lookup kostet damit keinen zusätzlichen Dienst, sondern eine zusätzliche
Bedingung in einer Abfrage, die sowieso stattfindet.

Das Cookie enthält einen Token aus 32 Byte `crypto/rand`, base64-kodiert. Gespeichert wird
nicht der Token, sondern sein SHA-256-Hash. Wer die Datenbank abzieht — über ein Backup,
eine SQL-Injection oder einen Blick ins Admin-Tool — bekommt damit Hashes, mit denen sich
niemand anmelden kann.

Hier entsteht ein scheinbarer Widerspruch zum Passwort-Hashing weiter unten: Tokens werden
mit schlichtem SHA-256 ohne Salt gehasht, Passwörter mit Argon2id und Salt. Beides ist
Absicht, aus zwei Gründen:

1. Der Lookup muss deterministisch sein. Mit zufälligem Salt könnte man nicht nach dem Hash
   suchen, sondern müsste jede Zeile einzeln durchprobieren.
2. Argon2 ist absichtlich langsam, weil Passwörter kurz und erratbar sind. 32 zufällige
   Bytes sind nicht erratbar. Langsamkeit schützt hier vor nichts und kostet nur Rechenzeit
   auf jedem Request.

Der Zugriff läuft über ein Interface, nicht direkt über die Tabelle:

```go
type SessionStore interface {
    Create(ctx context.Context, s Session) error
    Get(ctx context.Context, tokenHash []byte) (Session, error)
    Touch(ctx context.Context, tokenHash []byte, expiresAt time.Time) error
    Delete(ctx context.Context, tokenHash []byte) error
    DeleteByUser(ctx context.Context, userID uuid.UUID) error
}
```

Damit bleibt die Store-Entscheidung austauschbar. Stellt sich Redis eines Tages doch als
nötig heraus, ist das eine zweite Implementierung und kein Umbau. `DeleteByUser` steht
bewusst schon im Interface — das ist "auf allen Geräten abmelden", also genau die
Fähigkeit, wegen der die Token-Variante verworfen wurde.

Auch das gleitende Verlängern läuft über den Store und nicht per direktem SQL aus der
Middleware, sonst wäre die Store-Wahl an einer Stelle doch wieder festgeschrieben. Die
15-Minuten-Drosselung steckt bewusst **nicht** in `Touch`, sondern beim Aufrufer: die
Middleware hat die Session bereits aus `Get` und ruft `Touch` nur, wenn deren `expires_at`
mehr als 15 Minuten unter dem Maximum liegt. Damit bleibt der Store frei von
Ablauf-Politik, und ein Aufruf von `Touch` bedeutet immer genau einen Schreibvorgang.

### Laufzeit: 30 Tage, gleitend

Jede Aktivität verlängert die Session um 30 Tage. Der `UPDATE` läuft aber nur, wenn die
letzte Verlängerung mehr als 15 Minuten her ist. Ohne diese Regel wäre jeder Request ein
Schreibvorgang in der Datenbank — deutlich teurer als der Lookup und der einzige Punkt, an
dem die Store-Wahl überhaupt spürbar würde. Mit ihr bleibt es bei einem Schreibvorgang pro
Nutzer und Viertelstunde.

Die 30 Tage kommen aus dem Nutzungsmuster: eine Rechnungsanwendung wird nicht täglich
benutzt, sondern schubweise, oft nur zum Monatsende. Bei einer kürzeren Frist würde
praktisch jeder Besuch mit einem Login anfangen. Das begrenzt die Sitzungsdauer nur auf dem
Papier — in der Praxis führt es dazu, dass Passwörter im Browser gespeichert oder kürzer
gewählt werden. Das größere Zeitfenster für einen gestohlenen Token ist vertretbar, weil
genau dieser Fall beherrschbar ist: Zeile löschen, Sitzung tot. Dafür liegt der Zustand auf
dem Server.

Ohne die gleitende Verlängerung würde jemand nach 30 Tagen ab Login herausfliegen, zu einem
Zeitpunkt, der nichts mit seiner Arbeit zu tun hat — und je sporadischer die Nutzung, desto
wahrscheinlicher trifft es genau den Besuch, bei dem etwas erledigt werden soll.

Aufgeräumt wird ohne Hintergrundprozess: eine abgelaufene Zeile wird gelöscht, wenn sie
beim Lookup angefasst wird, und beim erfolgreichen Login werden zusätzlich die abgelaufenen
Sessions desselben Nutzers mitgelöscht. Wer wiederkommt, räumt hinter sich auf. Übrig
bleiben Zeilen von Konten, die nie wieder auftauchen — bei diesen Datenmengen kein Grund
für eine Goroutine, die beim Herunterfahren sauber beendet werden muss und deren Fehler
niemand sieht. Für die Korrektheit spielt das ohnehin keine Rolle, der Lookup prüft
`expires_at > now()`.

### Cookie-Attribute und CSRF

`HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`.

Ein CSRF-Token gibt es nicht. `SameSite=Lax` sorgt dafür, dass der Browser das Cookie bei
Requests von fremden Seiten nicht mitschickt, außer bei einer normalen Navigation per
`GET` — und die API ändert bei `GET` nichts. Zusätzlich nimmt sie ausschließlich JSON an;
ein Formular auf einer fremden Seite kann keinen `POST` mit `Content-Type:
application/json` absetzen, ohne vorher am CORS-Preflight zu scheitern.

Diese Entscheidung ist neu zu prüfen, sobald eine der Bedingungen wegfällt:

- Die API nimmt Formular-Submits an (`application/x-www-form-urlencoded`,
  `multipart/form-data`)
- Ein Frontend auf einer anderen Origin greift zu, das Cookie braucht `SameSite=None`
- Ein `GET`-Endpoint ändert Zustand

### Passwörter: Argon2id

| Parameter | Wert |
|---|---|
| Speicher | 19 MiB (19456 KiB) |
| Iterationen | 2 |
| Parallelität | 1 |
| Salt | 16 Byte aus `crypto/rand`, pro Passwort neu |
| Ausgabe | 32 Byte |

Die Werte stammen aus dem [OWASP Password Storage Cheat
Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).
Argon2id selbst ist in [RFC 9106](https://www.rfc-editor.org/rfc/rfc9106.html)
spezifiziert; dessen eigene Empfehlungen liegen höher (2 GiB, ersatzweise 64 MiB), zielen
aber auf Umgebungen, in denen so viel Speicher pro Login-Vorgang verfügbar ist. Für ein
Backend, das mehrere Logins gleichzeitig bedienen können soll, ist der OWASP-Wert der
brauchbare Kompromiss.

Nicht bcrypt: es lässt sich nicht speicherintensiv konfigurieren und ist damit auf GPUs
deutlich besser angreifbar, außerdem schneidet es Passwörter nach 72 Byte ab.

Gespeichert wird im PHC-Format, also inklusive der verwendeten Parameter:

```text
$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
```

Damit lassen sich die Kosten später erhöhen, ohne alle Bestandshashes ungültig zu machen:
beim nächsten erfolgreichen Login wird mit den neuen Parametern neu gehasht.

### Bausteine

`golang.org/x/crypto/argon2` und `crypto/rand`. Keine Auth-Bibliothek und kein Framework.
Was hier zu bauen ist — Passwort prüfen, Zufallstoken erzeugen, Zeile schreiben, Cookie
setzen, Middleware — ist überschaubar, und selbst geschrieben ist es nachvollziehbar
statt konfiguriert. Das ist bei Sicherheitscode sonst kein gutes Argument; es gilt hier,
weil die kritischen Teile (Hashing, Zufall) aus der Standardbibliothek und aus
`x/crypto` kommen und eben nicht selbst gebaut werden.

## Konsequenzen

**Jeder authentifizierte Request kostet einen Lookup.** Grob 0,3 bis 0,8 ms inklusive
Roundtrip; das ist gegenüber der Token-Variante der Preis für den Widerruf. Zum Vergleich:
der Netzwerkweg vom Browser liegt bei 20 bis 80 ms.

**Login dauert rund 45 ms**, davon etwa 40 ms Argon2. Das ist gewollt und darf nicht
"optimiert" werden.

**Es kommt eine Migration** für die Session-Tabelle. Ein Hintergrundprozess wird nicht
gebraucht.

**Ein Deployment loggt niemanden aus**, weil der Zustand in Postgres liegt und nicht im
Arbeitsspeicher.

**Ein nativer mobiler Client würde diese Entscheidung neu aufwerfen.** Cookies sind dort
unhandlicher, und der Widerruf müsste anders gelöst werden. Solange es beim Browser-Client
bleibt, stellt sich die Frage nicht.

## Offene Punkte

**Rate-Limiting am Login.** Argon2id verteuert Angriffe auf eine *gestohlene Datenbank*.
Gegen jemanden, der einfach tausende Passwörter gegen den laufenden Login-Endpoint schickt,
hilft es nicht — im Gegenteil, jeder Versuch kostet den Server 40 ms Rechenzeit, was den
Endpoint auch zum Hebel für eine Überlastung macht. Braucht eine eigene Maßnahme (Sperre
pro Konto und pro IP mit ansteigender Wartezeit) und ein eigenes Ticket.

**OAuth-Anbieter.** Ob GitHub-, Google- oder Microsoft-Login dazukommt, ist eine
Produktentscheidung und wird hier nicht vorweggenommen. Die Entscheidung für
server-seitige Sessions steht dem nicht im Weg — ein OAuth-Callback endet ebenfalls damit,
dass eine Session angelegt wird.

**Passwort vergessen.** Nicht Teil dieser Entscheidung, braucht aber E-Mail-Versand und
damit einen eigenen Zuschnitt.

**Mandantenmodell.** Ob ein Nutzer zu genau einer Firma gehört oder zu mehreren, ist in #36
offen und wird hier nicht vorweggenommen. Auf diese Entscheidung wirkt es sich nicht aus,
wohl aber auf den Inhalt der Session: bei mehreren Firmen pro Nutzer muss sie zusätzlich
festhalten, in welcher Firma gerade gearbeitet wird. Braucht eine eigene ADR.
