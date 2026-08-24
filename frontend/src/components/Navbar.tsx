import { useState, useRef, useEffect } from 'react'
import { Link, useLocation } from 'react-router'
import {
    HouseIcon,
    UserIcon,
    ReceiptIcon,
    UsersIcon,
    CaretUpDownIcon,
} from '@phosphor-icons/react'
import { useAuth } from '../auth/AuthContext.ts'
import styles from './Navbar.module.css'
import logo from '../assets/VentoryGo.png'

// Kundenübersicht, Rechnungsübersicht und Teams & Mitarbeiter haben noch keine
// eigenen Routen -- die Links bleiben bewusst inert (kein <Link>, kein Klick-
// Handler), bis die jeweiligen Seiten existieren. Nur "Home" navigiert wirklich.
const NAV_ITEMS = [
    { label: 'Home', icon: HouseIcon, path: '/dashboard' },
    { label: 'Kundenübersicht', icon: UserIcon, path: null },
    { label: 'Rechnungsübersicht', icon: ReceiptIcon, path: null },
    { label: 'Teams & Mitarbeiter', icon: UsersIcon, path: null },
] as const

export function Navbar() {
    const { user, logout } = useAuth()
    const location = useLocation()
    const [menuOpen, setMenuOpen] = useState(false)
    const menuRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        if (!menuOpen) return

        function handleClickOutside(event: MouseEvent) {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                setMenuOpen(false)
            }
        }
        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
    }, [menuOpen])

    return (
        <nav className={styles.navbar}>
            <div className={styles.left}>
                <img src={logo} alt="VentoryGo" className={styles.logo} />

                <div className={styles.navItems}>
                    {NAV_ITEMS.map(({ label, icon: Icon, path }) => {
                        const active = path !== null && location.pathname === path
                        const content = (
                            <>
                                <Icon size={18} weight={active ? 'fill' : 'regular'} />
                                <span>{label}</span>
                            </>
                        )

                        return path ? (
                            <Link key={label} to={path} className={`${styles.navItem} ${active ? styles.navItemActive : ''}`}>
                                {content}
                            </Link>
                        ) : (
                            <span key={label} className={styles.navItem} aria-disabled="true">
                                {content}
                            </span>
                        )
                    })}
                </div>
            </div>

            <div className={styles.account} ref={menuRef}>
                <button type="button" className={styles.accountButton} onClick={() => setMenuOpen((open) => !open)}>
                    <span className={styles.accountEmail}>{user?.email}</span>
                    <CaretUpDownIcon size={16} />
                </button>

                {menuOpen && (
                    <div className={styles.dropdown}>
                        <button type="button" className={styles.dropdownItem} disabled>
                            Einstellungen
                        </button>
                        <button type="button" className={styles.dropdownItemDanger} onClick={() => void logout()}>
                            Abmelden
                        </button>
                    </div>
                )}
            </div>
        </nav>
    )
}