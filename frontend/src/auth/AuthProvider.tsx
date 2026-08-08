import { useEffect, useState, type ReactNode } from 'react'
import { AuthContext, type User } from './AuthContext'
import { api, setUnauthorizedHandler } from './api'

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null)
    const [loading, setLoading] = useState(true)

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
        setUser(me)
        return me
    }

    useEffect(() => {
        let cancelled = false

        refreshMe()
            .catch(() => {
                if (!cancelled) setUser(null)
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
        await api<User>('/api/auth/register', {
            method: 'POST',
            body: { email, password },
        })
        // Registrierung meldet nicht automatisch an — kein Cookie wird gesetzt
        // (Handler.Register ruft service.Register, nicht service.Login).
        // Ticket-AC "kein zweiter manueller Login" ist damit NICHT erfüllt durch
        // den aktuellen Backend-Code. Sobald bestätigt/geändert: hier anpassen.
        await login(email, password)
    }

    async function logout() {
        await api('/api/auth/logout', { method: 'POST' })
        setUser(null)
    }

    return (
        <AuthContext.Provider value={{ user, loading, login, register, logout }}>
            {children}
        </AuthContext.Provider>
    )
}