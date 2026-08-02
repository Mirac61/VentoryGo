import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router'
import { useAuth } from '../auth/AuthContext.ts'

export function RequireAuth({ children }: { children: ReactNode }) {
    const { user, loading } = useAuth()
    const location = useLocation()

    // Solange der erste /api/auth/me-Aufruf läuft, wird weder Login noch
    // geschützter Inhalt gerendert — sonst blitzt /login beim Reload eines
    // angemeldeten Nutzers kurz auf.
    if (loading) {
        return null
    }

    if (!user) {
        return <Navigate to="/login" replace state={{ from: location }} />
    }

    return <>{children}</>
}