import { useEffect, useRef, useState } from 'react'
import { getDocument, GlobalWorkerOptions } from 'pdfjs-dist'
import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import styles from './PdfCanvas.module.css'

GlobalWorkerOptions.workerSrc = workerUrl


const MAX_CANVAS_PIXELS = 32_000_000

const RESIZE_DEBOUNCE_MS = 150
const MIN_WIDTH_CHANGE_PX = 2

interface PdfCanvasProps {
    data: ArrayBuffer | null
    onPageCount?: (count: number) => void
}


export function PdfCanvas({ data, onPageCount }: PdfCanvasProps) {
    const containerRef = useRef<HTMLDivElement>(null)
    const [width, setWidth] = useState(0)
    const onPageCountRef = useRef(onPageCount)

    useEffect(() => {
        onPageCountRef.current = onPageCount
    }, [onPageCount])

    useEffect(() => {
        const container = containerRef.current
        if (!container) return

        let timer: number | undefined
        let isFirstMeasurement = true

        const observer = new ResizeObserver((entries) => {
            const next = Math.round(entries[0].contentRect.width)

            if (isFirstMeasurement) {
                isFirstMeasurement = false
                setWidth(next)
                return
            }
            window.clearTimeout(timer)
            timer = window.setTimeout(() => {
                setWidth((previous) => (Math.abs(previous - next) < MIN_WIDTH_CHANGE_PX ? previous : next))
            }, RESIZE_DEBOUNCE_MS)
        })
        observer.observe(container)

        return () => {
            window.clearTimeout(timer)
            observer.disconnect()
        }
    }, [])

    useEffect(() => {
        const container = containerRef.current
        if (!data || !container || width === 0) return

        let cancelled = false
        const loadingTask = getDocument({ data: new Uint8Array(data.slice(0)) })

        async function render(target: HTMLDivElement) {
            const pdf = await loadingTask.promise
            const pixelRatio = window.devicePixelRatio || 1
            const canvases: HTMLCanvasElement[] = []

            for (let pageNumber = 1; pageNumber <= pdf.numPages; pageNumber++) {
                const page = await pdf.getPage(pageNumber)
                if (cancelled) return

                const fitScale = width / page.getViewport({ scale: 1 }).width
                const cssViewport = page.getViewport({ scale: fitScale })
                const maxSharpness = Math.sqrt(MAX_CANVAS_PIXELS / (cssViewport.width * cssViewport.height))
                const viewport = page.getViewport({ scale: fitScale * Math.min(pixelRatio, maxSharpness) })

                const canvas = document.createElement('canvas')
                canvas.width = Math.floor(viewport.width)
                canvas.height = Math.floor(viewport.height)
                canvas.className = styles.page

                await page.render({ canvas, viewport }).promise
                canvases.push(canvas)
            }

            if (cancelled) return
            target.replaceChildren(...canvases)
            onPageCountRef.current?.(pdf.numPages)
        }

        render(container).catch((error: unknown) => {
            if (!cancelled) console.error('PDF render failed', error)
        })

        return () => {
            cancelled = true
            void loadingTask.destroy()
        }
    }, [data, width])

    return <div ref={containerRef} className={styles.container} />
}