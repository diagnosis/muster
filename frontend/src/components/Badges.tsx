// src/components/Badges.tsx
import type {Outing} from "@/types.ts";
import styles from "@/components/Badges.module.css"

interface BadgesProps {
  outing: Outing
}

export function Badges({outing}: BadgesProps){
    const seat_cost = `$${(outing.cost_per_seat_cents / 100).toFixed(2)}/seat`
    return  <div className={styles.badges}>
        {outing.status==="cancelled" && <span className={`${styles.badge} ${styles.cancelled}`}>cancelled</span>}
        <span className={styles.badge}>{outing.difficulty}</span>
        <span className={styles.badge}>{outing.pace}</span>
        {outing.cost_per_seat_cents > 0 && <span className={`${styles.badge} ${styles.badgeCost}`}>{seat_cost}</span>}
    </div>

}