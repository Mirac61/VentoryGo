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

// "Gespeichert" must not hide that something is missing on the server.
function saveLabel(saveState: SaveState, rejectedLabels: string[], incompleteRowCount: number): string {
    if (rejectedLabels.length > 0) return `Nicht gespeichert: ${rejectedLabels.join(', ')}`
    if (saveState === 'saved' && incompleteRowCount > 0) {
        const rows =
            incompleteRowCount === 1
                ? '1 unvollständiger Eintrag'
                : `${incompleteRowCount} unvollständige Einträge`
        return `Gespeichert · ${rows} nicht gespeichert`
    }
    return SAVE_LABEL[saveState]
}

interface EditorTopbarProps {
    saveState: SaveState
    // Sections the backend refused, already translated for display.
    rejectedLabels: string[]
    // Line items that are not saved yet because they are incomplete or invalid.
    incompleteRowCount: number
    readOnly: boolean
    canPreview: boolean
    onSave: () => void
    onPreview: () => void
}

export function EditorTopbar({
    saveState,
    rejectedLabels,
    incompleteRowCount,
    readOnly,
    canPreview,
    onSave,
    onPreview,
}: EditorTopbarProps) {
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
                {saveLabel(saveState, rejectedLabels, incompleteRowCount)}
            </span>

            {/* The issue flow (confirmation dialog, POST /issue) is still open. */}
            <button type="button" className={styles.release} disabled title="Folgt als Nächstes">
                Rechnung ausstellen
            </button>
        </header>
    )
}