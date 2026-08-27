//src/components/OutingCard.tsx
import type {Outing} from "@/types.ts";
import styles from '@/components/OutingCard.module.css'
import {Badges} from "@/components/Badges.tsx";
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

    return <div className={styles.card}>
            <h2 className={styles.title}>{outing.title}</h2>
            <p className={styles.metaLine}>{outing.destination}</p>
            <p className={styles.metaLine}>{outing.meet_label}</p>
            <p className={styles.metaLine}>{starts_at_date} - {starts_at_time}</p>
            <Badges outing={outing}/>
    </div>

}