// src/components/Badges.tsx
import type { Outing } from '@/types.ts'
import styles from '@/components/Badges.module.css'

interface BadgesProps {
  outing: Outing
}

export function Badges({ outing }: BadgesProps) {
  const seat_cost = `$${(outing.cost_per_seat_cents / 100).toFixed(2)}/seat`
  return (
    <div className={styles.badges}>
      {outing.status === 'cancelled' && (
        <span data-testid="outing-cancelled" className={`${styles.badge} ${styles.cancelled}`}>
          cancelled
        </span>
      )}
      <span data-testid="outing-terrain" className={styles.badge}>
        <span className={styles.badgeLabel}>terrain</span> {outing.difficulty}
      </span>
      <span data-testid="outing-pace" className={styles.badge}>
        <span className={styles.badgeLabel}>pace</span> {outing.pace}
      </span>
      {outing.cost_per_seat_cents > 0 && (
        <span data-testid="outing-cost" className={`${styles.badge} ${styles.badgeCost}`}>
          {seat_cost}
        </span>
      )}
    </div>
  )
}
