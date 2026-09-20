import { ArrowUpRightIcon } from '@phosphor-icons/react'
import styles from './StatTile.module.css'

interface StatTileProps {
    label: string
    value: number
    loading: boolean
}

// Ersetzt die bisherige StatCard (Streifen in einem gemeinsamen statsGrid)
// durch eine eigenstaendige Karte, wie im Redesign-Mockup. Der Pfeil-Button
// ist rein dekorativ -- es gibt noch keine gefilterte Rechnungsansicht, zu
// der er verlinken koennte (Rechnungsübersicht hat noch keine Route).
export function StatTile({ label, value, loading }: StatTileProps) {
    return (
        <div className={styles.tile}>
            <div className={styles.header}>
                <span className={styles.label}>{label}</span>
                <span className={styles.arrowCircle} aria-hidden="true">
                    <ArrowUpRightIcon size={16} weight="bold" />
                </span>
            </div>
            <div className={styles.value}>{loading ? '–' : value}</div>
        </div>
    )
}