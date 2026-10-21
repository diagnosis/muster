import {Modal} from '@/components/Modal'
import type {ListedHiker} from '@/types'
import {initial} from '@/lib/format'
import styles from './WaitingListModal.module.css'

export function WaitingListModal({waiting, onClose}: {waiting: ListedHiker[], onClose: () => void}) {
    return (
        <Modal title={`Waiting (${waiting.length})`} onClose={onClose}>
            <p className={styles.rule}>Drivers go first, then in the order people asked. The host accepts from the top.</p>
            <ol className={styles.list}>
                {waiting.map((w, i) =>
                    <li key={i} className={w.is_me ? `${styles.row} ${styles.me}` : styles.row}
                        aria-current={w.is_me || undefined}>
                        <span className={styles.position}>{i + 1}</span>
                        <span className={styles.avatar} aria-hidden="true">{initial(w.name)}</span>
                        <span className={styles.name}>
                            {w.name}{w.is_me && <span className={styles.you}> (you)</span>}
                        </span>
                        {w.role === 'driver' &&
                            <span className={styles.driverTag}>Driver · {w.seats_offered} {w.seats_offered === 1 ? 'seat' : 'seats'}</span>}
                        {w.guests > 0 &&
                            <span className={styles.guests}>+{w.guests} {w.guests === 1 ? 'guest' : 'guests'}</span>}
                    </li>)}
            </ol>
        </Modal>
    )
}