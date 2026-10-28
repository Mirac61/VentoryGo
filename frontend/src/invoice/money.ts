// Betraege kommen aus dem Backend als Integer in Cent, nicht als Euro-Float
// (siehe backend/README.md, "Geldbetraege: Cent als Integer"). Deshalb wird
// hier nur fuer die Anzeige formatiert -- gerechnet wird weiter mit Cent.

export function formatCentsDisplay(cents: number, currency: string): string {
    const sign = cents < 0 ? '-' : ''
    const absolute = Math.abs(cents)
    const whole = String(Math.floor(absolute / 100)).replace(/\B(?=(\d{3})+(?!\d))/g, '.')
    const fraction = String(absolute % 100).padStart(2, '0')
    return `${sign}${whole},${fraction} ${currency === 'EUR' ? '€' : currency}`
}

const COUNT_FORMAT = new Intl.NumberFormat('de-DE')

export function formatCount(count: number): string {
    return COUNT_FORMAT.format(count)
}
