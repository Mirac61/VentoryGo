import { useState } from 'react'
import { Link, useLocation } from 'react-router'
import {
    CirclesFourIcon,
    UserIcon,
    ReceiptIcon,
    UsersFourIcon,
    SidebarSimpleIcon,
} from '@phosphor-icons/react'
import styles from './Sidebar.module.css'
import logo from '../assets/VentoryGo.png'

// Kundenübersicht, Rechnungsübersicht und Teams & Mitarbeiter haben noch keine
// eigenen Routen -- wie schon in der bisherigen Navbar bleiben sie inert
// (kein <Link>, kein Klick-Handler), bis die jeweiligen Seiten existieren.
// Nur Dashboard navigiert wirklich.
const NAV_ITEMS = [
    { label: 'Dashboard', icon: CirclesFourIcon, path: '/dashboard' },
    { label: 'Kundenübersicht', icon: UserIcon, path: null },
    { label: 'Rechnungsübersicht', icon: ReceiptIcon, path: null },
    { label: 'Teams & Mitarbeiter', icon: UsersFourIcon, path: null },
] as const

export function Sidebar() {
    const location = useLocation()
    // Rein clientseitiger UI-Zustand -- kein Speichern der Praeferenz ueber
    // einen Reload hinaus, das war in der bisherigen Navbar auch nicht der Fall.
    const [collapsed, setCollapsed] = useState(false)

    return (
        <aside className={`${styles.sidebar} ${collapsed ? styles.sidebarCollapsed : ''}`}>
            <div className={styles.header}>
                <Link to="/dashboard" className={styles.brand}>
                    <img src={logo} alt="VentoryGo" className={styles.logo} />
                    {!collapsed}
                </Link>
                <button
                    type="button"
                    className={styles.collapseButton}
                    onClick={() => setCollapsed((value) => !value)}
                    aria-label={collapsed ? 'Sidebar ausklappen' : 'Sidebar einklappen'}
                    aria-expanded={!collapsed}
                >
                    <SidebarSimpleIcon size={20} />
                </button>
            </div>

            <nav className={styles.nav}>
                {NAV_ITEMS.map(({ label, icon: Icon, path }) => {
                    const active = path !== null && location.pathname === path
                    const content = (
                        <>
                            <Icon size={20} weight={active ? 'fill' : 'regular'} />
                            {!collapsed && <span>{label}</span>}
                        </>
                    )

                    return path ? (
                        <Link
                            key={label}
                            to={path}
                            className={`${styles.navItem} ${active ? styles.navItemActive : ''}`}
                            title={collapsed ? label : undefined}
                        >
                            {content}
                        </Link>
                    ) : (
                        <span
                            key={label}
                            className={styles.navItem}
                            aria-disabled="true"
                            title={collapsed ? label : undefined}
                        >
                            {content}
                        </span>
                    )
                })}
            </nav>
        </aside>
    )
}