//src/components/OutingCard.tsx
import type {Outing} from "@/types.ts";
import styles from '@/components/OutingCard.module.css'
import {Badges} from "@/components/Badges.tsx";
import {outingRange} from "@/utils/date.ts";
interface OutingCardProps {
   outing: Outing
}

export function OutingCard({outing}: OutingCardProps){


    return <div className={styles.card}>
            <h2 className={styles.title}>{outing.title}</h2>
            <p className={styles.metaLine}>{outing.destination}</p>
            <p className={styles.metaLine}>{outing.meet_label}</p>
            <p className={styles.metaLine}>{outingRange(outing.starts_at, outing.ends_at)}</p>
            <Badges outing={outing}/>
    </div>

}