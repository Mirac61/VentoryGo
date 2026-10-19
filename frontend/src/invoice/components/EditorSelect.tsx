import { useId, type SelectHTMLAttributes } from 'react'
import styles from './EditorField.module.css'

interface EditorSelectOption {
    value: string
    label: string
}

interface EditorSelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
    label: string
    options: EditorSelectOption[]
}

export function EditorSelect({ label, options, id, disabled, ...selectProps }: EditorSelectProps) {
    const generatedId = useId()
    const fieldId = id ?? generatedId
    const selectClass = [styles.input, disabled && styles.readOnly].filter(Boolean).join(' ')

    return (
        <div className={styles.field}>
            <div className={styles.labelRow}>
                <label htmlFor={fieldId}>{label}</label>
            </div>
            <select id={fieldId} className={selectClass} disabled={disabled} {...selectProps}>
                {options.map((option) => (
                    <option key={option.value} value={option.value}>
                        {option.label}
                    </option>
                ))}
            </select>
        </div>
    )
}