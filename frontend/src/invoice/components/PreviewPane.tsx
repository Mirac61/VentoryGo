import { useEffect, useRef, useState } from 'react'
import { MagnifyingGlassMinusIcon, MagnifyingGlassPlusIcon } from '@phosphor-icons/react'
import { PdfCanvas } from '../../components/PdfCanvas'
import styles from './PreviewPane.module.css'

// 100 % is "fit to the pane", capped at SHEET_WIDTH_PX so the sheet does not
// grow absurdly wide on large monitors.
const ZOOM_STEPS = [0.5, 0.75, 1, 1.25, 1.5, 2]
const FIT_INDEX = 2
const LAST_INDEX = ZOOM_STEPS.length - 1
const SHEET_WIDTH_PX = 860

// A wheel notch or a trackpad pinch fires many events; one zoom step per
// interval keeps the zoom from racing through all levels at once.
const WHEEL_STEP_INTERVAL_MS = 120

interface PreviewPaneProps {
    pdfBytes: ArrayBuffer | null
    error: Error | null
}

export function PreviewPane({ pdfBytes, error }: PreviewPaneProps) {
    const [zoomIndex, setZoomIndex] = useState(FIT_INDEX)
    const [pageCount, setPageCount] = useState(0)
    const stageRef = useRef<HTMLDivElement>(null)
    const lastWheelStep = useRef(0)
    const zoom = ZOOM_STEPS[zoomIndex]

    useEffect(() => {
        const stage = stageRef.current
        if (!stage) return

        function handleWheel(event: WheelEvent) {
            if (!event.ctrlKey) return
            event.preventDefault()

            const now = Date.now()
            if (now - lastWheelStep.current < WHEEL_STEP_INTERVAL_MS) return
            lastWheelStep.current = now

            setZoomIndex((index) => (event.deltaY < 0 ? Math.min(LAST_INDEX, index + 1) : Math.max(0, index - 1)))
        }

        stage.addEventListener('wheel', handleWheel, { passive: false })
        return () => stage.removeEventListener('wheel', handleWheel)
    }, [])

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
                    <span className={styles.zoomLabel} title="Strg + Mausrad zum Zoomen">
                        {Math.round(zoom * 100)} %
                    </span>
                    <button
                        type="button"
                        className={styles.iconButton}
                        aria-label="Vergrößern"
                        disabled={zoomIndex === LAST_INDEX}
                        onClick={() => setZoomIndex((index) => Math.min(LAST_INDEX, index + 1))}
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

            {error && pdfBytes && (
                <p className={styles.error} role="status">
                    Vorschau konnte nicht aktualisiert werden.
                </p>
            )}

            <div className={styles.stage} ref={stageRef}>
                {pdfBytes ? (
                    <div
                        className={styles.sheetColumn}
                        style={{ width: `${zoom * SHEET_WIDTH_PX}px` }}
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