// Dünner fetch-Wrapper für alle API-Aufrufe. Relative Pfade, same-origin
// dank Vite-Proxy (dev) bzw. gemeinsamem Ursprung (prod) — kein
// `credentials: "include"` nötig, das Session-Cookie geht automatisch mit.

export class ApiError extends Error {
    status: number
    fieldErrors?: Record<string, string>
    retryAfterSeconds?: number

    constructor(
        status: number,
        message: string,
        options?: { fieldErrors?: Record<string, string>; retryAfterSeconds?: number },
    ) {
        super(message)
        this.name = 'ApiError'
        this.status = status
        this.fieldErrors = options?.fieldErrors
        this.retryAfterSeconds = options?.retryAfterSeconds
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
    const { body, skipUnauthorizedRedirect, headers, ...rest } = options

    const response = await fetch(path, {
        ...rest,
        headers: {
            ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
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
            // Kein JSON-Body, egal.
        }

        const message = extractMessage(payload) ?? `Anfrage fehlgeschlagen (${response.status})`
        const fieldErrors = response.status === 422 ? extractFieldErrors(payload) : undefined
        const retryAfterHeader = response.headers.get('Retry-After')
        const retryAfterSeconds = retryAfterHeader ? Number(retryAfterHeader) : undefined

        throw new ApiError(response.status, message, {
            fieldErrors,
            retryAfterSeconds: Number.isFinite(retryAfterSeconds) ? retryAfterSeconds : undefined,
        })
    }

    if (response.status === 204) {
        return undefined as T
    }

    return (await response.json()) as T
}

function extractMessage(payload: unknown): string | undefined {
    if (payload && typeof payload === 'object' && 'error' in payload) {
        const value = (payload as { error?: unknown }).error
        if (typeof value === 'string') return value
    }
    return undefined
}

function extractFieldErrors(payload: unknown): Record<string, string> | undefined {
    if (payload && typeof payload === 'object' && 'fieldErrors' in payload) {
        const value = (payload as { fieldErrors?: unknown }).fieldErrors
        if (value && typeof value === 'object') {
            return value as Record<string, string>
        }
    }
    return undefined
}