import { useEffect, useRef, useState, type ReactNode } from 'react'
import { AuthContext, type User } from './AuthContext'
import { api, setUnauthorizedHandler } from './api'

// Konto wurde angelegt, der anschließende automatische Login ist aber
// fehlgeschlagen. Kein Registrierungsfehler — der Aufrufer sollte hier auf
// "bitte manuell anmelden" statt auf die üblichen Registrierungsfehler mappen.
export class PostRegisterLoginError extends Error {
    cause: unknown
    constructor(cause: unknown) {
        super('Konto wurde angelegt, automatische Anmeldung ist fehlgeschlagen')
        this.name = 'PostRegisterLoginError'
        this.cause = cause
    }
}

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null)
    const [loading, setLoading] = useState(true)

    // Zählt jeden erfolgreichen login()/refreshMe(). Damit erkennt das
    // anfängliche /me beim Mount, ob zwischenzeitlich (während es noch in
    // Flight war) bereits ein erfolgreicher Login passiert ist — und darf den
    // frischen User-State dann NICHT mit seinem eigenen (veralteten) 401
    // überschreiben.
    const authGeneration = useRef(0)

    useEffect(() => {
        // 401 aus einem beliebigen späteren Aufruf räumt den Kontext und schickt
        // zurück zu /login (Session mitten in der Sitzung abgelaufen).
        setUnauthorizedHandler(() => {
            setUser(null)
            if (window.location.pathname !== '/login') {
                window.location.assign('/login')
            }
        })
        return () => setUnauthorizedHandler(null)
    }, [])

    async function refreshMe() {
        // GET /api/auth/me liefert den User direkt (kein { user: ... }-Wrapper),
        // siehe Handler.Me: c.JSON(http.StatusOK, user).
        const me = await api<User>('/api/auth/me', { method: 'GET', skipUnauthorizedRedirect: true })
        authGeneration.current += 1
        setUser(me)
        return me
    }

    useEffect(() => {
        let cancelled = false
        const generationAtStart = authGeneration.current

        refreshMe()
            .catch(() => {
                // Nur räumen, wenn seitdem kein erfolgreicher Login/refreshMe
                // stattgefunden hat — sonst würde ein spätes 401 aus DIESEM Aufruf
                // eine inzwischen gültige, frischere Session wieder wegräumen.
                if (!cancelled && authGeneration.current === generationAtStart) {
                    setUser(null)
                }
            })
            .finally(() => {
                if (!cancelled) setLoading(false)
            })

        return () => {
            cancelled = true
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    async function login(email: string, password: string) {
        // Login liefert 204 ohne Body (siehe Handler.Login) — der Cookie ist
        // gesetzt, aber wer angemeldet ist, muss separat über /me geholt werden.
        await api<void>('/api/auth/login', {
            method: 'POST',
            body: { email, password },
        })
        await refreshMe()
    }

    async function register(email: string, password: string) {
        // Register liefert 201 + User direkt (kein Wrapper, kein Cookie/Login).
        // TODO Backend: Handler.Register sollte selbst einloggen (ein Request
        // statt zwei) — ADR/Ticket-AC "kein zweiter manueller Login" ist mit dem
        // jetzigen Code nur durch diesen Workaround erfüllt.
        await api<User>('/api/auth/register', {
            method: 'POST',
            body: { email, password },
        })

        // Konto existiert ab hier garantiert. Schlägt NUR der Login danach fehl
        // (Netzwerk, Server-Hiccup), ist das kein Registrierungsfehler — der
        // Aufrufer bekommt das über einen eigenen Error-Typ mitgeteilt, statt
        // fälschlich "Registrierung fehlgeschlagen" zu zeigen.
        try {
            await login(email, password)
        } catch (err) {
            throw new PostRegisterLoginError(err)
        }
    }

    async function logout() {
        await api('/api/auth/logout', { method: 'POST' })
        authGeneration.current += 1
        setUser(null)
    }

    return (
        <AuthContext.Provider value={{ user, loading, login, register, logout }}>
            {children}
        </AuthContext.Provider>
    )
}