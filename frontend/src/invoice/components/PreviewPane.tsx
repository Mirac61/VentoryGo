import { useState } from 'react'
import { MagnifyingGlassMinusIcon, MagnifyingGlassPlusIcon } from '@phosphor-icons/react'
import { PdfCanvas } from '../../components/PdfCanvas'
import styles from './PreviewPane.module.css'

// 100 % is "fit to the pane", capped at SHEET_WIDTH_PX so the sheet does not
// grow absurdly wide on large monitors.
const ZOOM_STEPS = [0.5, 0.75, 1, 1.25, 1.5, 2]
const FIT_INDEX = 2
const SHEET_WIDTH_PX = 860

interface PreviewPaneProps {
    pdfBytes: ArrayBuffer | null
    error: Error | null
}

export function PreviewPane({ pdfBytes, error }: PreviewPaneProps) {
    const [zoomIndex, setZoomIndex] = useState(FIT_INDEX)
    const [pageCount, setPageCount] = useState(0)
    const zoom = ZOOM_STEPS[zoomIndex]

    return (
        <section className={styles.pane} aria-label="Live-Vorschau der Rechnung">
            <div className={styles.toolbar}>
                <div className={styles.zoom}>
                    <button
                        type="button"
                        className={styles.iconButton}
                        aria-label="Verkleinern"
                        disabled={zoomIndex === 0}
                        onClick={() => setZoomIndex((index) => Math.max(0, index - 1))}
                    >
                        <MagnifyingGlassMinusIcon size={18} />
                    </button>
                    <span className={styles.zoomLabel}>{Math.round(zoom * 100)} %</span>
                    <button
                        type="button"
                        className={styles.iconButton}
                        aria-label="Vergrößern"
                        disabled={zoomIndex === ZOOM_STEPS.length - 1}
                        onClick={() => setZoomIndex((index) => Math.min(ZOOM_STEPS.length - 1, index + 1))}
                    >
                        <MagnifyingGlassPlusIcon size={18} />
                    </button>
                    <button type="button" className={styles.fit} onClick={() => setZoomIndex(FIT_INDEX)}>
                        Anpassen
                    </button>
                </div>
                <span className={styles.pages}>
                    {pageCount > 0 ? `${pageCount} ${pageCount === 1 ? 'Seite' : 'Seiten'}` : ''}
                </span>
            </div>

            {/* The old PDF stays visible when a refresh fails. */}
            {error && pdfBytes && (
                <p className={styles.error} role="status">
                    Vorschau konnte nicht aktualisiert werden.
                </p>
            )}

            <div className={styles.stage}>
                {pdfBytes ? (
                    <div
                        className={styles.sheetColumn}
                        style={{ width: `min(${zoom * 100}%, ${zoom * SHEET_WIDTH_PX}px)` }}
                    >
                        <PdfCanvas data={pdfBytes} onPageCount={setPageCount} />
                    </div>
                ) : (
                    <p className={styles.empty}>{error ? 'Vorschau nicht verfügbar.' : 'Vorschau wird geladen …'}</p>
                )}
            </div>
        </section>
    )
}