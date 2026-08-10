import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../auth/api'
import { PostRegisterLoginError } from '../auth/AuthProvider'
import { registerErrorMessage } from '../auth/errorMessages'
import { AuthCard } from '../auth/components/AuthCard'
import { Field } from '../auth/components/Field'
import formStyles from '../auth/components/authForm.module.css'

const MIN_PASSWORD_LENGTH = 12

// Backend liefert bei 422 validator-Tags, keine Texte (z.B. "email", "min").
// Übersetzung hier zentral, damit Field-Komponenten nur Anzeigetext sehen.
function fieldErrorMessage(field: string, tag: string): string {
    if (field === 'email' && tag === 'email') return 'Bitte eine gültige E-Mail-Adresse angeben'
    if (field === 'email' && tag === 'required') return 'E-Mail ist erforderlich'
    if (field === 'password' && tag === 'min') return `Passwort muss mindestens ${MIN_PASSWORD_LENGTH} Zeichen lang sein`
    if (field === 'password' && tag === 'required') return 'Passwort ist erforderlich'
    return 'Ungültige Eingabe'
}

export default function Register() {
    const { register } = useAuth()
    const navigate = useNavigate()

    const [email, setEmail] = useState('')
    const [password, setPassword] = useState('')
    const [passwordConfirm, setPasswordConfirm] = useState('')
    const [submitting, setSubmitting] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
    const [passwordConfirmError, setPasswordConfirmError] = useState<string | undefined>()

    function validateClientSide(): boolean {
        // Nur der Client-only-Check (Passwörter stimmen überein) — die
        // Mindestlänge prüft der Server ohnehin per 422, doppelte Prüfung hier
        // würde nur zu abweichender Fehlertext-Quelle führen.
        if (passwordConfirm !== password) {
            setPasswordConfirmError('Passwörter stimmen nicht überein')
            return false
        }
        setPasswordConfirmError(undefined)
        return true
    }

    async function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault()
        if (submitting) return

        setError(null)
        setFieldErrors({})

        if (!validateClientSide()) return

        setSubmitting(true)
        try {
            await register(email, password)
            navigate('/dashboard', { replace: true })
        } catch (err) {
            if (err instanceof PostRegisterLoginError) {
                // Konto existiert bereits — kein Registrierungsfehler. Nutzer soll
                // sich manuell anmelden, nicht denken, das Konto sei nicht entstanden.
                setError('Konto wurde erstellt, die automatische Anmeldung ist aber fehlgeschlagen. Bitte melde dich manuell an.')
            } else if (err instanceof ApiError && err.status === 422 && err.fieldErrors) {
                const messages: Record<string, string> = {}
                for (const [field, tag] of Object.entries(err.fieldErrors)) {
                    messages[field] = fieldErrorMessage(field, tag)
                }
                setFieldErrors(messages)
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
                    error={passwordConfirmError}
                    required
                />

                <button type="submit" className={formStyles.submit} disabled={submitting}>
                    {submitting ? 'Wird registriert …' : 'Registrieren'}
                </button>
            </form>
        </AuthCard>
    )
}