import { Link } from 'react-router'
import { ArrowClockwiseIcon, ArrowCounterClockwiseIcon } from '@phosphor-icons/react'
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

            {/* Undo/redo is not built yet: visible but disabled, like the other stubs. */}
            <button type="button" className={styles.iconButton} disabled aria-label="Rückgängig" title="Noch nicht verfügbar">
                <ArrowCounterClockwiseIcon size={18} />
            </button>
            <button type="button" className={styles.iconButton} disabled aria-label="Wiederholen" title="Noch nicht verfügbar">
                <ArrowClockwiseIcon size={18} />
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
