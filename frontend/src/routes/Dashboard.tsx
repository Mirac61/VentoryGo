import { useAuth } from '../auth/AuthContext.ts'

export function Dashboard() {
    const { user, logout } = useAuth()

    return (
        <div style={{ padding: 32 }}>
            <h1>Dashboard</h1>
            <p>Angemeldet als {user?.email}</p>
            <button type="button" onClick={() => void logout()}>
                Abmelden
            </button>
        </div>
    )
}