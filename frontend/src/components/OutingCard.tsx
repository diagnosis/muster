//src/components/OutingCard.tsx
import type {Outing} from "../types.ts";
import styles from './OutingCard.module.css'
interface OutingCardProps {
   outing: Outing
}

export function OutingCard({outing}: OutingCardProps){

    const date = new Date(outing.starts_at)
    const starts_at_date =
        date.toLocaleDateString('en-US', {
                weekday: 'short',
                month:'short',
                day:'numeric'
            })
    const starts_at_time=
        date.toLocaleTimeString('en-US', {
                hour:'numeric',
                minute:'2-digit'
            })
    const seat_cost = `$${(outing.cost_per_seat_cents / 100).toFixed(2)}/seat `
    return <div className={styles.card}>
            <h2 className={styles.title}>{outing.title}</h2>
            <p className={styles.metaLine}>{outing.destination}</p>
            <p className={styles.metaLine}>{outing.meet_label}</p>
            <p className={styles.metaLine}>{starts_at_date} - {starts_at_time}</p>
            <div className={styles.badges}>
                {outing.status==="cancelled" && <span className={`${styles.badge} ${styles.cancelled}`}>cancelled</span>}
                <span className={styles.badge}>{outing.difficulty}</span>
                <span className={styles.badge}>{outing.pace}</span>
                {outing.cost_per_seat_cents > 0 && <span className={`${styles.badge} ${styles.badgeCost}`}>{seat_cost}</span>}
            </div>
    </div>

}