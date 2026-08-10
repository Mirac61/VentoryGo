import type { ReactNode } from 'react'
import { WarningCircleIcon } from '@phosphor-icons/react'
import styles from './AuthCard.module.css'
import logo from '../../assets/VentoryGo.png'

interface AuthCardProps {
    title: string
    subtitle: string
    error: string | null
    children: ReactNode
    footer: ReactNode
}

export function AuthCard({ title, subtitle, error, children, footer }: AuthCardProps) {
    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <div className={styles.left}>
                    <div className={styles.brand}>
                        {<img src={logo} width="299" height="58" alt="VentoryGo" className={styles.logo} />}
                    </div>

                    <div className={styles.heading}>
                        <h1 className={styles.title}>{title}</h1>
                        <p className={styles.subtitle}>{subtitle}</p>
                    </div>
                </div>

                <div className={styles.right}>
                    {error && (
                        <div className={styles.errorBar} role="alert">
                            <WarningCircleIcon size={20} weight="fill" />
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