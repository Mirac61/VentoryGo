import { ApiError } from './api'

// Handler.Login gibt bei falschen Daten 401 + {"error": "invalid email or
// password"} zurück — bewusst ohne Unterschied zwischen "Mail existiert
// nicht" und "Passwort falsch", das übernimmt das UI hier 1:1.
export function loginErrorMessage(error: unknown): string {
    if (error instanceof ApiError && error.status === 401) {
        return 'E-Mail oder Passwort ist falsch'
    }
    return 'Anmeldung fehlgeschlagen. Bitte versuche es erneut.'
}

// Handler.Register gibt bei Duplikat 409 + {"error": "email already
// registered"} zurück.
export function registerErrorMessage(error: unknown): string {
    if (error instanceof ApiError && error.status === 409) {
        return 'E-Mail ist bereits registriert'
    }
    return 'Registrierung fehlgeschlagen. Bitte versuche es erneut.'
}

// Rate-Limiting existiert laut Backend-Doku noch nicht — hier bewusst keine
// 429-Behandlung, bis der echte Endpunkt/Response bekannt ist.