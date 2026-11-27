//src/components/OutingCard.tsx
import type { Outing } from '@/types.ts'
import styles from '@/components/OutingCard.module.css'
import { Badges } from '@/components/Badges.tsx'
import { outingRange } from '@/utils/date.ts'
import { CalendarIcon, FlagIcon, PinIcon } from '@/components/Icons.tsx'
import { CapacityLine } from '@/components/CapacityLine.tsx'
interface OutingCardProps {
  outing: Outing
  label?: string
}

export function OutingCard({ outing, label }: OutingCardProps) {
  const live = outing.phase === 'in_progress'
  const past = outing.phase === 'past'
  const titleId = `outing-${outing.id}-title`
  return (
    <article
      aria-labelledby={titleId}
      className={`${styles.card} ${live ? styles.cardLive : ''} ${past ? styles.cardPast : ''}`}
    >
      {(label || live) && (
        <div className={styles.topRow}>
          {label && <span className={styles.label}>{label}</span>}
          {live && (
            <span data-testid="outing-live" className={styles.live}>
              Happening now
            </span>
          )}
        </div>
      )}
      <h2 id={titleId} className={styles.title}>
        {outing.title}
      </h2>
      <ul className={styles.meta}>
        <li data-testid="outing-date" className={`${styles.metaItem} ${styles.metaStrong}`}>
          <CalendarIcon size={16} /> {outingRange(outing.starts_at, outing.ends_at)}
        </li>
        <li data-testid="outing-destination" className={styles.metaItem}>
          <PinIcon size={16} /> {outing.destination}
        </li>
        <li data-testid="outing-meet" className={styles.metaItem}>
          <FlagIcon size={16} /> {outing.meet_label}
        </li>
      </ul>
      {outing.capacity && <CapacityLine capacity={outing.capacity} maxSize={outing.max_size} />}
      <Badges outing={outing} />
    </article>
  )
}
