import { useEffect, useState, type ReactNode } from 'react'
import { AuthContext, type User } from './AuthContext.ts'
import { api, setUnauthorizedHandler } from './api.ts'

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

    useEffect(() => {
        let cancelled = false

        api<User>('/api/auth/me', { skipUnauthorizedRedirect: true })
            .then((me) => {
                if (!cancelled) setUser(me)
            })
            .catch(() => {
                if (!cancelled) setUser(null)
            })
            .finally(() => {
                if (!cancelled) setLoading(false)
            })

        return () => {
            cancelled = true
        }
    }, [])

    async function login(email: string, password: string) {
        const me = await api<User>('/api/auth/login', {
            method: 'POST',
            body: { email, password },
        })
        setUser(me)
    }

    async function register(name: string, email: string, password: string) {
        const me = await api<User>('/api/auth/register', {
            method: 'POST',
            body: { name, email, password },
        })
        setUser(me)
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