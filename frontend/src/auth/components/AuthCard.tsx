import type { ReactNode } from 'react'
import styles from './AuthCard.module.css'
import logo from '../../assets/VentoryGo.png'

interface AuthCardProps {
    title: string
    subtitle?: string
    error: string | null
    children: ReactNode
    footer: ReactNode
}

// Logo sitzt jetzt auf Seitenebene (oben links), nicht mehr in der Card -
// entspricht dem neuen Redesign (siehe Anmeldung.png / Registrierung.png).
// Die Linien im Hintergrund sind rein dekorativ (aria-hidden) und bilden
// das Drei-Spalten-Raster aus dem Mockup nach: zwei volle vertikale Linien,
// dazu die Unterkante der Kopfzeile in der linken/rechten Spalte (nicht in
// der mittleren, wo Logo/Karte sitzen).
const LEFT_COLUMN = '22.3%'
const RIGHT_COLUMN = '78.2%'

export function AuthCard({ title, error, children, footer }: AuthCardProps) {
    return (
        <div className={styles.page}>
            <div className={styles.gridLines} aria-hidden="true">
                <div className={styles.gridLineVertical} style={{ left: LEFT_COLUMN }} />
                <div className={styles.gridLineVertical} style={{ left: RIGHT_COLUMN }} />
            </div>

            {/* Die Kopfzeile traegt ihre eigene Unterkante als Linie - sie
                skaliert damit automatisch mit Logo-Groesse/Padding statt an
                einem aus dem Mockup abgelesenen Pixelwert zu haengen. */}
            <div className={styles.headerRow}>
                <div className={styles.headerSegment} style={{ width: LEFT_COLUMN }} />
                <div className={styles.pageLogo}>
                    <img src={logo} alt="VentoryGo" className={styles.pageLogoImg} />
                </div>
                <div className={styles.headerSegment} style={{ flex: 1 }} />
            </div>

            <div className={styles.center}>
                <div className={styles.card}>
                    <h1 className={styles.title}>{title}</h1>
                    {error && (
                        <div className={styles.errorText} role="alert">
                            <span>{error}</span>
                        </div>
                    )}

                    {children}
                    {footer}
                </div>
            </div>
        </div>
    )
}