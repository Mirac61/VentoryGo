// Das Backend liefert Zeitstempel als ISO-Strings; leere Daten kommen als
// "0001-01-01T00:00:00Z" (siehe detailsForm.dateOnly). Ausgabe als deutsches
// Datum mit fuehrenden Nullen -- ohne die stehen in einer Liste "1.10.2026" und
// "30.11.2026" untereinander und die Spalte wirkt schief.
//
// Formatiert wird bewusst nur der Datumsteil: "2026-10-01T00:00:00Z" ist der
// 1. Oktober, nicht der 30. September. `new Date(iso)` wuerde den Zeitstempel
// in die lokale Zeitzone umrechnen und westlich von UTC einen Tag zurueck
// springen -- das Datum im Backend ist aber bereits ein Kalendertag.
const ISO_DATE = /^(\d{4})-(\d{2})-(\d{2})/

export function formatIsoDate(iso: string): string {
    if (!iso || iso.startsWith('0001')) return ''
    const match = ISO_DATE.exec(iso)
    if (match === null) return ''

    const [, year, month, day] = match
    // Gueltigkeit pruefen, damit z. B. "2026-02-31" nicht als 31.02. erscheint.
    const date = new Date(Date.UTC(Number(year), Number(month) - 1, Number(day)))
    if (
        date.getUTCFullYear() !== Number(year) ||
        date.getUTCMonth() !== Number(month) - 1 ||
        date.getUTCDate() !== Number(day)
    ) {
        return ''
    }

    return `${day}.${month}.${year}`
}
