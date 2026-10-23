export function isValidEmail(value: string): boolean {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
}

// The backend wants IBANs uppercase and without spaces.
export function normalizeIban(value: string): string {
    return value.replace(/\s+/g, '').toUpperCase()
}

// Close enough to the backend check to avoid sending values it would reject:
// shape, 15 to 34 characters, mod-97 checksum. The backend additionally knows
// the exact length per country.
export function isPlausibleIban(value: string): boolean {
    const iban = normalizeIban(value)
    if (!/^[A-Z]{2}\d{2}[A-Z0-9]{11,30}$/.test(iban)) return false

    const rearranged = iban.slice(4) + iban.slice(0, 4)
    let remainder = 0
    for (const char of rearranged) {
        const digits = /\d/.test(char) ? char : String(char.charCodeAt(0) - 55)
        for (const digit of digits) {
            remainder = (remainder * 10 + Number(digit)) % 97
        }
    }
    return remainder === 1
}
