import { Link } from 'react-router'
import type { SaveState } from '../useInvoiceAutosave'
import styles from './EditorTopbar.module.css'

const SAVE_LABEL: Record<SaveState, string> = {
    idle: '',
    saving: 'Speichert …',
    saved: 'Gespeichert',
    error: 'Speichern fehlgeschlagen',
    frozen: 'Nicht mehr bearbeitbar',
}

interface EditorTopbarProps {
    saveState: SaveState
    readOnly: boolean
    canPreview: boolean
    onSave: () => void
    onPreview: () => void
}

export function EditorTopbar({ saveState, readOnly, canPreview, onSave, onPreview }: EditorTopbarProps) {
    return (
        <header className={styles.topbar}>
            <Link to="/dashboard" className={styles.brand}>
                VentoryGo
            </Link>

            <button
                type="button"
                className={styles.action}
                onClick={onSave}
                disabled={readOnly || saveState === 'saving'}
            >
                Speichern
            </button>
            <button type="button" className={styles.action} onClick={onPreview} disabled={!canPreview}>
                Vorschau
            </button>

            <span className={styles.saveState} role="status">
                {SAVE_LABEL[saveState]}
            </span>

            {/* The issue flow (confirmation dialog, POST /issue) is the next slice. */}
            <button type="button" className={styles.release} disabled title="Folgt als Nächstes">
                Rechnung ausstellen
            </button>
        </header>
    )
}
