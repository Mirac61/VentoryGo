// Dünner fetch-Wrapper für alle API-Aufrufe. Same-origin dank Vite-Proxy
// (dev) bzw. gemeinsamem Ursprung (prod). `credentials: 'include'` explizit
// gesetzt, macht die Absicht sichtbar und schadet same-origin nicht.

export class ApiError extends Error {
    status: number
    fieldErrors?: Record<string, string>

    constructor(status: number, message: string, fieldErrors?: Record<string, string>) {
        super(message)
        this.name = 'ApiError'
        this.status = status
        this.fieldErrors = fieldErrors
    }
}

// Wird von AuthProvider gesetzt. Erlaubt api() bei 401 den Kontext zu räumen
// und auf /login umzuleiten, ohne dass jeder Aufrufer das selbst prüfen muss.
let onUnauthorized: (() => void) | null = null

export function setUnauthorizedHandler(handler: (() => void) | null) {
    onUnauthorized = handler
}

interface ApiOptions extends Omit<RequestInit, 'body'> {
    body?: unknown
    // 401 auf /api/auth/me beim Start ist erwartet (noch nicht angemeldet),
    // soll dort also nicht den globalen Redirect auslösen.
    skipUnauthorizedRedirect?: boolean
}

export async function api<T>(path: string, options: ApiOptions = {}): Promise<T> {
    const { body, skipUnauthorizedRedirect, headers, method, ...rest } = options

    const isWrite = method !== undefined && method.toUpperCase() !== 'GET'

    const response = await fetch(path, {
        ...rest,
        method,
        credentials: 'include',
        headers: {
            ...(isWrite ? { 'Content-Type': 'application/json' } : {}),
            ...headers,
        },
        body: body !== undefined ? JSON.stringify(body) : undefined,
    })

    if (response.status === 401 && !skipUnauthorizedRedirect) {
        onUnauthorized?.()
    }

    if (!response.ok) {
        let payload: unknown = null
        try {
            payload = await response.json()
        } catch {
            // Kein JSON-Body, egal (z.B. bindJSON schlägt vor dem JSON-Encoding fehl).
        }

        if (response.status === 422) {
            const fieldErrors = extractFieldErrors(payload)
            throw new ApiError(422, 'Eingabe ungültig', fieldErrors)
        }

        const message = extractFlatMessage(payload) ?? `Anfrage fehlgeschlagen (${response.status})`
        throw new ApiError(response.status, message)
    }

    if (response.status === 204) {
        return undefined as T
    }

    return (await response.json()) as T
}

function extractFlatMessage(payload: unknown): string | undefined {
    if (payload && typeof payload === 'object' && 'message' in payload) {
        const value = (payload as { message?: unknown }).message
        if (typeof value === 'string') return value
    }
    return undefined
}

function extractFieldErrors(payload: unknown): Record<string, string> | undefined {
    if (payload && typeof payload === 'object' && 'fields' in payload) {
        const value = (payload as { fields?: unknown }).fields
        if (value && typeof value === 'object') {
            return value as Record<string, string>
        }
    }
    return undefined
}