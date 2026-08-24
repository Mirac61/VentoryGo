import { PlusIcon } from '@phosphor-icons/react'
import styles from './NewInvoiceCard.module.css'

interface NewInvoiceCardProps {
    onClick: () => void
}

// Navigiert noch nirgendwo hin -- der Rechnungseditor existiert noch nicht.
// onClick liegt trotzdem schon als Prop an, damit der Aufrufer (Dashboard)
// entscheidet, was "neu" bedeutet, sobald die Route da ist.
export function NewInvoiceCard({ onClick }: NewInvoiceCardProps) {
    return (
        <button type="button" className={styles.card} onClick={onClick}>
            <span className={styles.iconCircle}>
                <PlusIcon size={20} weight="bold" />
            </span>
            <span className={styles.label}>Neue Rechnung</span>
        </button>
    )
}