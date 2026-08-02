import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../auth/api'
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
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

    async function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault()
        if (submitting) return

        setSubmitting(true)
        setError(null)
        setFieldErrors({})

        try {
            await login(email, password)
            const redirectTo = (location.state as { from?: Location })?.from?.pathname ?? '/dashboard'
            navigate(redirectTo, { replace: true })
        } catch (err) {
            if (err instanceof ApiError && err.status === 422 && err.fieldErrors) {
                setFieldErrors(err.fieldErrors)
            } else {
                setError(loginErrorMessage(err))
            }
        } finally {
            setSubmitting(false)
        }
    }

    return (
        <AuthCard
            title="Willkommen!"
            subtitle="Melde dich an, um fortzufahren."
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
                    error={fieldErrors.email}
                    required
                />
                <Field
                    label="Passwort"
                    type="password"
                    autoComplete="current-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    error={fieldErrors.password}
                    required
                />

                <a href="#" className={formStyles.forgotPassword}>
                    Passwort vergessen?
                </a>

                <button type="submit" className={formStyles.submit} disabled={submitting}>
                    {submitting ? 'Wird angemeldet …' : 'Anmelden'}
                </button>
            </form>
        </AuthCard>
    )
}