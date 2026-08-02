import { ApiError } from './api'

// Copy exakt aus dem Ticket. 401 bewusst ohne Hinweis, welches der beiden
// Felder falsch war.
export function loginErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      return 'E-Mail oder Passwort ist falsch'
    }
    if (error.status === 429) {
      return retryAfterMessage(error.retryAfterSeconds)
    }
  }
  return 'Anmeldung fehlgeschlagen. Bitte versuche es erneut.'
}

export function registerErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 409) {
      return 'E-Mail ist bereits registriert'
    }
    if (error.status === 429) {
      return retryAfterMessage(error.retryAfterSeconds)
    }
  }
  return 'Registrierung fehlgeschlagen. Bitte versuche es erneut.'
}

function retryAfterMessage(retryAfterSeconds: number | undefined): string {
  if (!retryAfterSeconds) {
    return 'Zu viele Versuche. Bitte versuche es später erneut.'
  }
  if (retryAfterSeconds < 60) {
    return `Zu viele Versuche. Bitte warte ${retryAfterSeconds} Sekunden.`
  }
  const minutes = Math.ceil(retryAfterSeconds / 60)
  return `Zu viele Versuche. Bitte warte ${minutes} Minute${minutes === 1 ? '' : 'n'}.`
}
