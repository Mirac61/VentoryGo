import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../auth/api'
import { registerErrorMessage } from '../auth/errorMessages'
import { AuthCard } from '../auth/components/AuthCard'
import { Field } from '../auth/components/Field'
import formStyles from '../auth/components/authForm.module.css'

const MIN_PASSWORD_LENGTH = 12

export default function Register() {
    const { register } = useAuth()
    const navigate = useNavigate()

    const [name, setName] = useState('')
    const [email, setEmail] = useState('')
    const [password, setPassword] = useState('')
    const [passwordConfirm, setPasswordConfirm] = useState('')
    const [submitting, setSubmitting] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

    function validateClientSide(): Record<string, string> {
        const errors: Record<string, string> = {}

        if (password.length < MIN_PASSWORD_LENGTH) {
            errors.password = `Passwort muss mindestens ${MIN_PASSWORD_LENGTH} Zeichen lang sein`
        }
        if (passwordConfirm !== password) {
            errors.passwordConfirm = 'Passwörter stimmen nicht überein'
        }

        return errors
    }

    async function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault()
        if (submitting) return

        setError(null)

        // Client-seitige Prüfung ist nur ein UX-Vorteil (früheres Feedback) —
        // der Server bleibt trotzdem die Instanz, die letztlich entscheidet.
        const clientErrors = validateClientSide()
        if (Object.keys(clientErrors).length > 0) {
            setFieldErrors(clientErrors)
            return
        }
        setFieldErrors({})

        setSubmitting(true)
        try {
            await register(name, email, password)
            navigate('/dashboard', { replace: true })
        } catch (err) {
            if (err instanceof ApiError && err.status === 422 && err.fieldErrors) {
                setFieldErrors(err.fieldErrors)
            } else {
                setError(registerErrorMessage(err))
            }
        } finally {
            setSubmitting(false)
        }
    }

    return (
        <AuthCard
            title="Willkommen!"
            subtitle="Registriere, um fortzufahren."
            error={error}
            footer={
                <p className={formStyles.footer}>
                    Du hast schon ein Konto? <Link to="/login">Melde dich an!</Link>
                </p>
            }
        >
            <form onSubmit={handleSubmit} noValidate>
                <Field
                    label="Name"
                    type="text"
                    autoComplete="name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    error={fieldErrors.name}
                    required
                />
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
                    hint={`mind. ${MIN_PASSWORD_LENGTH} Zeichen`}
                    type="password"
                    autoComplete="new-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    error={fieldErrors.password}
                    required
                />
                <Field
                    label="Passwort wiederholen"
                    type="password"
                    autoComplete="new-password"
                    value={passwordConfirm}
                    onChange={(e) => setPasswordConfirm(e.target.value)}
                    error={fieldErrors.passwordConfirm}
                    required
                />

                <button type="submit" className={formStyles.submit} disabled={submitting}>
                    {submitting ? 'Wird registriert …' : 'Registrieren'}
                </button>
            </form>
        </AuthCard>
    )
}