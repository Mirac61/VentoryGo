import { PlusIcon } from '@phosphor-icons/react'
import styles from './NewInvoiceButton.module.css'

interface NewInvoiceButtonProps {
    onClick: () => void
}

// Kompakte Variante von NewInvoiceCard fuer die Leiste neben "Rechnungen" im
// neuen Dashboard-Layout. NewInvoiceCard (die große gestrichelte Karte) bleibt
// unangetastet, falls sie anderswo noch gebraucht wird -- onClick ist bewusst
// noch ein no-op beim Aufrufer, der Rechnungseditor existiert noch nicht.
export function NewInvoiceButton({ onClick }: NewInvoiceButtonProps) {
    return (
        <button type="button" className={styles.button} onClick={onClick}>
            <PlusIcon size={16} weight="bold" />
            Neue Rechnung
        </button>
    )
}