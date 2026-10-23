import { useId, type TextareaHTMLAttributes } from 'react'
import styles from './EditorField.module.css'

interface EditorTextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
    label: string
    hint?: string
}

export function EditorTextArea({ label, hint, id, readOnly, ...textareaProps }: EditorTextAreaProps) {
    const generatedId = useId()
    const fieldId = id ?? generatedId
    const textareaClass = [styles.input, styles.textarea, readOnly && styles.readOnly].filter(Boolean).join(' ')

    return (
        <div className={styles.field}>
            <div className={styles.labelRow}>
                <label htmlFor={fieldId}>{label}</label>
                {hint && <span className={styles.hint}>{hint}</span>}
            </div>
            <textarea id={fieldId} className={textareaClass} readOnly={readOnly} {...textareaProps} />
        </div>
    )
}