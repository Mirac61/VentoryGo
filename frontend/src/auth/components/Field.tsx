import { useId, type InputHTMLAttributes } from 'react'
import { WarningCircleIcon } from '@phosphor-icons/react'
import styles from './Field.module.css'

interface FieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string
  hint?: string
  error?: string
}

export function Field({ label, hint, error, id, ...inputProps }: FieldProps) {
  const generatedId = useId()
  const fieldId = id ?? generatedId

  return (
    <div className={styles.field}>
      <div className={styles.labelRow}>
        <label htmlFor={fieldId}>{label}</label>
        {hint && <span className={styles.hint}>{hint}</span>}
      </div>
      <input
        id={fieldId}
        className={error ? styles.inputError : styles.input}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${fieldId}-error` : undefined}
        {...inputProps}
      />
      {error && (
        <div className={styles.error} id={`${fieldId}-error`}>
          <WarningCircleIcon size={16} weight="fill" />
          <span>{error}</span>
        </div>
      )}
    </div>
  )
}
