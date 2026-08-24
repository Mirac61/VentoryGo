import { useInvoices } from '../invoice/useInvoices'
import { useDashboardStats } from '../invoice/dashboardStats'
import { Navbar } from '../components/Navbar'
import { NewInvoiceCard } from '../components/NewInvoiceCard'
import styles from './Dashboard.module.css'

// "Dein Unternehmen GmbH" ist ein Platzhalter -- es gibt noch kein Firmenprofil-
// Feld auf User (siehe AuthContext.ts: nur id, email, createdAt). Sobald #36
// (Firmenprofil) steht, kommt der Name von dort statt hier fest zu stehen.
const COMPANY_NAME_PLACEHOLDER = 'Dein Unternehmen GmbH'

export function Dashboard() {
    const { invoices, fetching, fetchResult, refetchInvoices } = useInvoices()
    const stats = useDashboardStats(invoices)

    return (
        <div className={styles.page}>
            <Navbar />

            <div className={styles.content}>
                <h1 className={styles.companyName}>{COMPANY_NAME_PLACEHOLDER}</h1>

                <NewInvoiceCard onClick={() => { /* Rechnungseditor existiert noch nicht */ }} />

                <section className={styles.statsSection}>
                    <div className={styles.statsHeader}>
                        <h2 className={styles.statsTitle}>Statistiken</h2>
                        <button type="button" className={styles.refreshButton} onClick={refetchInvoices} disabled={fetching}  >
                            {fetching ? 'Lädt …' : 'Aktualisieren'}
                        </button>
                    </div>

                    {fetchResult && <p className={styles.errorText}>Fehler: {fetchResult.message}</p>}

                    {!fetchResult && (
                        <div className={styles.statsGrid}>
                            <StatCard label="Überfällig" value={stats.ueberfaellig} accent="danger" loading={fetching} />
                            <StatCard label="Offene Rechnungen" value={stats.offeneRechnungen} accent="primary" loading={fetching} />
                            <StatCard label="Bezahlt (diesen Monat)" value={stats.bezahltDiesenMonat} accent="success" loading={fetching} />
                            <StatCard label="Entwürfe" value={stats.entwuerfe} accent="neutral" loading={fetching} />
                        </div>
                    )}
                </section>
            </div>
        </div>
    )
}

interface StatCardProps {
    label: string
    value: number
    accent: 'danger' | 'primary' | 'success' | 'neutral'
    loading: boolean
}

function StatCard({ label, value, accent, loading }: StatCardProps) {
    return (
        <div className={styles.statCard}>
            <div className={styles.statValue}>{loading ? '–' : value}</div>
            <div className={styles.statLabel}>{label}</div>
            <div className={`${styles.statBar} ${styles[`statBar${capitalize(accent)}`]}`} />
        </div>
    )
}

function capitalize(word: string): string {
    return word.charAt(0).toUpperCase() + word.slice(1)
}