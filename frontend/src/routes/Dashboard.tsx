import { useInvoices } from '../invoice/useInvoices'
import { useDashboardStats } from '../invoice/dashboardStats'
import { Sidebar } from '../components/Sidebar'
import { Navbar } from '../components/Navbar'
import { NewInvoiceButton } from '../components/NewInvoiceButton'
import { InvoiceRow } from '../components/InvoiceRow'
import { StatTile } from '../components/StatTile'
import styles from './Dashboard.module.css'
import { useState } from 'react'
import { useNavigate } from 'react-router'
import { createDraft } from '../invoice/createDraft'

// Kein Vorname auf User (siehe AuthContext.ts: nur id, email, createdAt) --
// "Max" im Mockup ist ein Platzhalter, bis ein Profilfeld existiert.
const FIRST_NAME_PLACEHOLDER = 'Max'


export function Dashboard() {
    const { invoices, fetching, fetchResult, refetchInvoices } = useInvoices()
  const stats = useDashboardStats(invoices)

  const navigate = useNavigate()
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  
  async function handleNewInvoice() {
      if (creating) return
      setCreating(true)
      setCreateError(null)
      try {
          const draft = await createDraft()
          navigate(`/invoices/${draft.id}`)
      } catch {
          setCreateError('Rechnung konnte nicht angelegt werden. Bitte versuche es erneut.')
          setCreating(false)
      }
  }
    return (
        <div className={styles.page}>
            <Sidebar />

            <div className={styles.main}>
                <Navbar title="Dashboard" />

                <div className={styles.content}>
                    <div className={styles.column}>
                        <h1 className={styles.greeting}>
                            Servus, <span>{FIRST_NAME_PLACEHOLDER}!</span>
                        </h1>

                        <div className={styles.listHeader}>
                            <h2 className={styles.listTitle}>Rechnungen</h2>
                            <NewInvoiceButton onClick={handleNewInvoice} disabled={creating} />
                        </div>
                        {createError && <p className={styles.errorText}>{createError}</p>}

                        <div className={styles.listCard}>
                            {fetchResult && <p className={styles.errorText}>Fehler: {fetchResult.message}</p>}
                            {!fetchResult && !fetching && invoices.length === 0 && (
                                <p className={styles.emptyText}>Noch keine Rechnungen vorhanden.</p>
                            )}
                            {!fetchResult && invoices.map((invoice) => (
                                <InvoiceRow key={invoice.id} invoice={invoice} />
                            ))}
                        </div>
                    </div>

                    <div className={styles.column}>
                        <div className={styles.statsHeader}>
                            <h2 className={styles.statsTitle}>Statistiken</h2>
                            <button type="button" className={styles.refreshButton} onClick={refetchInvoices} disabled={fetching}>
                                {fetching ? 'Lädt …' : 'Aktualisieren'}
                            </button>
                        </div>

                        {/* Platzhalter -- das Diagramm folgt in einem spaeteren Schritt. */}
                        <div className={styles.chartPlaceholder} />

                        <div className={styles.statsGrid}>
                            <StatTile label="Offene Rechnungen" value={stats.offeneRechnungen} loading={fetching} />
                            <StatTile label="Bezahlte Rechnungen" value={stats.bezahltDiesenMonat} loading={fetching} />
                            <StatTile label="Überfällig" value={stats.ueberfaellig} loading={fetching} />
                            <StatTile label="im Entwurf" value={stats.entwuerfe} loading={fetching} />
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}