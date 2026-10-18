import { useId, type InputHTMLAttributes } from 'react'
import styles from './EditorField.module.css'

interface EditorFieldProps extends InputHTMLAttributes<HTMLInputElement> {
    label: string
    hint?: string
    error?: string
}

export function EditorField({ label, hint, error, id, readOnly, ...inputProps }: EditorFieldProps) {
    const generatedId = useId()
    const fieldId = id ?? generatedId
    const inputClass = [styles.input, error && styles.invalid, readOnly && styles.readOnly]
        .filter(Boolean)
        .join(' ')

    return (
        <div className={styles.field}>
            <div className={styles.labelRow}>
                <label htmlFor={fieldId}>{label}</label>
                {hint && <span className={styles.hint}>{hint}</span>}
            </div>
            <input
                id={fieldId}
                className={inputClass}
                readOnly={readOnly}
                aria-invalid={error ? true : undefined}
                aria-describedby={error ? `${fieldId}-error` : undefined}
                {...inputProps}
            />
            {error && (
                <p className={styles.error} id={`${fieldId}-error`}>
                    {error}
                </p>
            )}
        </div>
    )
}
