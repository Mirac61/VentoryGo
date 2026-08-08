import { createContext, useContext } from 'react'

// Entspricht backend/internal/auth/user.go — kein "name"-Feld vorhanden.
export interface User {
    id: string
    email: string
    createdAt: string
}

export interface AuthContextValue {
    user: User | null
    loading: boolean
    login: (email: string, password: string) => Promise<void>
    register: (email: string, password: string) => Promise<void>
    logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth(): AuthContextValue {
    const context = useContext(AuthContext)
    if (!context) {
        throw new Error('useAuth muss innerhalb eines AuthProvider verwendet werden')
    }
    return context
}