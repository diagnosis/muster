//src/components/OutingCard.tsx
import type {Outing} from "@/types.ts";
import styles from '@/components/OutingCard.module.css'
import {Badges} from "@/components/Badges.tsx";
import {outingRange} from "@/utils/date.ts";
import {CalendarIcon, FlagIcon, PinIcon} from "@/components/Icons.tsx";
interface OutingCardProps {
   outing: Outing
    label?: string
}

export function OutingCard({outing, label}: OutingCardProps) {
    const live = outing.phase === "in_progress"
    const past = outing.phase === "past"
    return <div className={`${styles.card} ${live ? styles.cardLive : ""} ${past ? styles.cardPast : ""}`}>
        {(label || live) && <div className={styles.topRow}>
            {label && <span className={styles.label}>{label}</span>}
            {live && <span className={styles.live}>Happening now</span>}
        </div>}
        <h2 className={styles.title}>{outing.title}</h2>
        <ul className={styles.meta}>
            <li className={`${styles.metaItem} ${styles.metaStrong}`}>
                <CalendarIcon size={16}/> {outingRange(outing.starts_at, outing.ends_at)}
            </li>
            <li className={styles.metaItem}><PinIcon size={16}/> {outing.destination}</li>
            <li className={styles.metaItem}><FlagIcon size={16}/> {outing.meet_label}</li>
        </ul>
        <Badges outing={outing}/>
    </div>
}