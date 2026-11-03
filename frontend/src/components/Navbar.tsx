import styles from './Navbar.module.css'

export function Navbar({ title }: { title: string }) {
    return (
        <nav className={styles.navbar}>
            <h1 className={styles.title}>{title}</h1>
        </nav>
    )
}