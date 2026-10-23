import type { Contact, Issuer } from '../Invoice'
import { isPlausibleIban, isValidEmail } from '../validation'
import { EditorField } from './EditorField'
import styles from './CompanyCustomerTab.module.css'

interface CompanyCustomerTabProps {
    sender: Issuer
    recipient: Contact
    readOnly: boolean
    onSenderChange: (key: keyof Issuer, value: string) => void
    onRecipientChange: (key: keyof Contact, value: string) => void
}

const EMAIL_ERROR = 'Bitte eine gültige E-Mail-Adresse angeben'

export function CompanyCustomerTab({
    sender,
    recipient,
    readOnly,
    onSenderChange,
    onRecipientChange,
}: CompanyCustomerTabProps) {
    const senderField = (key: keyof Issuer, label: string, extra: { hint?: string; error?: string; type?: string } = {}) => (
        <EditorField
            label={label}
            value={sender[key]}
            readOnly={readOnly}
            autoComplete="off"
            onChange={(event) => onSenderChange(key, event.target.value)}
            {...extra}
        />
    )

    const recipientField = (key: keyof Contact, label: string, extra: { hint?: string; error?: string; type?: string } = {}) => (
        <EditorField
            label={label}
            value={recipient[key]}
            readOnly={readOnly}
            autoComplete="off"
            onChange={(event) => onRecipientChange(key, event.target.value)}
            {...extra}
        />
    )

    const senderEmailError = sender.email !== '' && !isValidEmail(sender.email.trim()) ? EMAIL_ERROR : undefined
    const recipientEmailError = recipient.email !== '' && !isValidEmail(recipient.email.trim()) ? EMAIL_ERROR : undefined
    const ibanError = sender.iban !== '' && !isPlausibleIban(sender.iban) ? 'Ungültige IBAN' : undefined

    return (
        <>
            <section className={styles.card}>
                <h2 className={styles.cardTitle}>Unternehmen</h2>
                <p className={styles.cardHelp}>Diese Angaben erscheinen als Absender auf der Rechnung.</p>

                {/* The backend has no logo upload yet. */}
                <button type="button" className={styles.logoButton} disabled title="Noch nicht verfügbar">
                    + Logo hochladen
                </button>

                {senderField('name', 'Firmenname')}
                {senderField('street', 'Straße und Nr.')}
                <div className={styles.rowZipCity}>
                    {senderField('zip', 'PLZ')}
                    {senderField('city', 'Ort')}
                </div>
                {senderField('country', 'Land')}
                {senderField('email', 'E-Mail', { type: 'email', error: senderEmailError })}
                {senderField('phone', 'Telefon', { type: 'tel', hint: 'optional' })}
                {senderField('vatId', 'Umsatzsteuer-IdNr.')}
                {senderField('taxNumber', 'Steuernummer', { hint: 'USt-IdNr. oder Steuernummer nötig' })}
            </section>

            <section className={styles.card}>
                <h2 className={styles.cardTitle}>Empfänger</h2>
                <p className={styles.cardHelp}>An wen geht die Rechnung?</p>

                {recipientField('name', 'Firmenname')}
                {recipientField('email', 'E-Mail', { type: 'email', error: recipientEmailError, hint: 'optional' })}
                {recipientField('street', 'Straße und Nr.')}
                <div className={styles.rowZipCity}>
                    {recipientField('zip', 'PLZ')}
                    {recipientField('city', 'Ort')}
                </div>
                {recipientField('country', 'Land')}
            </section>

            <section className={styles.card}>
                <h2 className={styles.cardTitle}>Bankverbindung</h2>
                <p className={styles.cardHelp}>Wohin soll der Rechnungsbetrag überwiesen werden?</p>

                {senderField('bankName', 'Bank')}
                {senderField('iban', 'IBAN', { error: ibanError })}
                {senderField('bic', 'BIC')}
            </section>
        </>
    )
}
