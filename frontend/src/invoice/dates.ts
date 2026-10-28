// Das Backend liefert Zeitstempel als ISO-Strings; leere Daten kommen als
// "0001-01-01T00:00:00Z" (siehe detailsForm.dateOnly). Ausgabe als deutsches
// Datum mit fuehrenden Nullen -- ohne die stehen in einer Liste "1.10.2026" und
// "30.11.2026" untereinander und die Spalte wirkt schief.
const DATE_FORMAT = new Intl.DateTimeFormat('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
})

export function formatIsoDate(iso: string): string {
    if (!iso || iso.startsWith('0001')) return ''
    const date = new Date(iso)
    if (Number.isNaN(date.getTime())) return ''
    return DATE_FORMAT.format(date)
}
