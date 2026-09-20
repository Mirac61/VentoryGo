import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router'
import { useAuth } from '../auth/AuthContext'
import { loginErrorMessage } from '../auth/errorMessages'
import { AuthCard } from '../auth/components/AuthCard'
import { Field } from '../auth/components/Field'
import formStyles from '../auth/components/authForm.module.css'

export default function Login() {
    const { login } = useAuth()
    const navigate = useNavigate()
    const location = useLocation()

    const [email, setEmail] = useState('')
    const [password, setPassword] = useState('')
    const [submitting, setSubmitting] = useState(false)
    const [error, setError] = useState<string | null>(null)

    async function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault()
        if (submitting) return

        setSubmitting(true)
        setError(null)

        try {
            await login(email, password)
            const redirectTo = (location.state as { from?: Location })?.from?.pathname ?? '/dashboard'
            navigate(redirectTo, { replace: true })
        } catch (err) {
            setError(loginErrorMessage(err))
        } finally {
            setSubmitting(false)
        }
    }

    return (
        <AuthCard
            title="Melde dich an, um fortzufahren"
            error={error}
            footer={
                <p className={formStyles.footer}>
                    Noch kein Konto? <Link to="/register">Erstelle dir eins!</Link>
                </p>
            }
        >
            <form onSubmit={handleSubmit} noValidate>
                <Field
                    label="E-Mail"
                    type="email"
                    autoComplete="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                />
                <Field
                    label="Passwort"
                    type="password"
                    autoComplete="current-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                />

                {/* Kein Passwort-vergessen-Flow im Backend (offener Punkt in
                    ADR 0001) - Link ist optisch da, aber deaktiviert. */}
                <span
                    className={formStyles.forgotPasswordDisabled}
                    aria-disabled="true"
                    title="Noch nicht verfügbar"
                >
                    Passwort vergessen?
                </span>

                <button type="submit" className={formStyles.submit} disabled={submitting}>
                    {submitting ? 'Wird angemeldet …' : 'Anmelden'}
                </button>

                {/* Kein OAuth-Handler im Backend (ADR 0001: "wird hier nicht
                    vorweggenommen") - Button ist sichtbar, aber deaktiviert. */}
                <button
                    type="button"
                    className={formStyles.oauthButton}
                    disabled
                    title="Noch nicht verfügbar"
                >

                    Mit GitHub anmelden
                </button>
            </form>
        </AuthCard>
    )
}