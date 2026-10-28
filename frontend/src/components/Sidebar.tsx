import { useEffect, useState } from 'react'
import { Link, useLocation } from 'react-router'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { ChevronsUpDown, LayoutDashboard, LogOut, PanelLeft, Receipt, Settings, Users, UsersRound } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import logo from '../assets/VentoryGo.png'
import styles from './Sidebar.module.css'

const NAV_ITEMS = [
    { label: 'Dashboard', icon: LayoutDashboard, path: '/dashboard' },
    { label: 'Kundenübersicht', icon: Users, path: null },
    { label: 'Rechnungsübersicht', icon: Receipt, path: null },
    { label: 'Teams & Mitarbeiter', icon: UsersRound, path: null },
] as const

export function Sidebar() {
    const { pathname } = useLocation()
    const { user, logout } = useAuth()
    // Auf schmalen Bildschirmen startet die Sidebar eingeklappt. Der Listener
    // haelt den Zustand aktuell, wenn das Fenster die 860px-Grenze ueberschreitet
    // -- sonst bliebe die einmal berechnete Startbreite stehen.
    const [collapsed, setCollapsed] = useState(() => window.matchMedia('(max-width: 860px)').matches)

    useEffect(() => {
        const query = window.matchMedia('(max-width: 860px)')
        const sync = () => setCollapsed(query.matches)
        sync()
        query.addEventListener('change', sync)
        return () => query.removeEventListener('change', sync)
    }, [])

    const initial = user?.email.charAt(0).toUpperCase() ?? '?'

    return (
        <aside className={styles.sidebar} data-collapsed={collapsed}>
            <div className={styles.header}>
                {!collapsed && (
                    <Link to="/dashboard" className={styles.brand}>
                        <img src={logo} alt="VentoryGo" className={styles.logo} />
                    </Link>
                )}
                <button
                    type="button"
                    className={styles.iconButton}
                    onClick={() => setCollapsed((value) => !value)}
                    aria-label={collapsed ? 'Sidebar ausklappen' : 'Sidebar einklappen'}
                    aria-expanded={!collapsed}
                >
                    <PanelLeft className={styles.icon} />
                </button>
            </div>

            <nav className={styles.nav} aria-label="Hauptnavigation">
                {NAV_ITEMS.map(({ label, icon: Icon, path }) => {
                    const active = path !== null && pathname === path
                    const content = (
                        <>
                            <Icon className={styles.icon} />
                            {!collapsed && <span className={styles.label}>{label}</span>}
                        </>
                    )

                    return path ? (
                        <Link
                            key={label}
                            to={path}
                            className={styles.navItem}
                            data-active={active}
                            data-label={label}
                            aria-label={collapsed ? label : undefined}
                            aria-current={active ? 'page' : undefined}
                        >
                            {content}
                        </Link>
                    ) : (
                        <span
                            key={label}
                            className={styles.navItem}
                            data-label={label}
                            aria-label={collapsed ? label : undefined}
                            aria-disabled="true"
                        >
                            {content}
                        </span>
                    )
                })}
            </nav>

            <div className={styles.footer}>
                <DropdownMenu.Root>
                    <DropdownMenu.Trigger asChild>
                        <button
                            type="button"
                            className={styles.account}
                            data-label="Konto"
                            aria-label={collapsed ? 'Konto-Menü' : undefined}
                        >
                            <span className={styles.avatar} aria-hidden="true">
                                {initial}
                            </span>
                            {!collapsed && (
                                <>
                                    <span className={styles.email}>{user?.email}</span>
                                    <ChevronsUpDown className={styles.chevron} />
                                </>
                            )}
                        </button>
                    </DropdownMenu.Trigger>

                    <DropdownMenu.Portal>
                        <DropdownMenu.Content
                            className={styles.menu}
                            side={collapsed ? 'right' : 'top'}
                            align={collapsed ? 'end' : 'start'}
                            sideOffset={8}
                        >
                            <DropdownMenu.Item className={styles.menuItem} disabled>
                                <Settings className={styles.menuIcon} />
                                Einstellungen
                            </DropdownMenu.Item>
                            <DropdownMenu.Item
                                className={`${styles.menuItem} ${styles.menuItemDanger}`}
                                onSelect={() => void logout()}
                            >
                                <LogOut className={styles.menuIcon} />
                                Abmelden
                            </DropdownMenu.Item>
                        </DropdownMenu.Content>
                    </DropdownMenu.Portal>
                </DropdownMenu.Root>
            </div>
        </aside>
    )
}