import { useState, useRef, useEffect } from 'react'
import { useAuth } from '../auth/AuthContext.ts'
import styles from './Navbar.module.css'
import placeholderAvatar from '../assets/avatar-placeholder.jpg'

interface NavbarProps {
    title: string
}

// Ersetzt die bisherige horizontale Navigation -- die Nav-Items sind in die
// Sidebar gewandert (siehe Sidebar.tsx). Was bleibt: der Seitentitel und das
// Konto-Menü mit Abmelden, das vorher hier am Rechts-Rand hing.
export function Navbar({ title }: NavbarProps) {
    const { logout } = useAuth()
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
            <h1 className={styles.title}>{title}</h1>

            <div className={styles.account} ref={menuRef}>
                {/* Platzhalterbild bis ein echtes Profilbild existiert (kein
                    entsprechendes Feld auf User, siehe AuthContext.ts). */}
                <button
                    type="button"
                    className={styles.accountButton}
                    onClick={() => setMenuOpen((open) => !open)}
                    aria-label="Konto-Menü"
                    aria-expanded={menuOpen}
                >
                    <img src={placeholderAvatar} alt="" className={styles.avatar} />

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