/* src/components/CapacityLine.tsx */

import type {Capacity} from "@/types.ts";
import styles from "@/components/CapacityLine.module.css";
import {capacityLine} from "@/lib/format.ts";

interface CapacityLineProps {
    capacity: Capacity
    maxSize: number
}

export function CapacityLine({capacity, maxSize}: CapacityLineProps) {
    const {text, tone} = capacityLine(capacity, maxSize)

    const pct = tone === 'open'
        ? Math.round((capacity.people_count / (capacity.people_count + capacity.spots_left)) * 100)
        : 100
    return (
        <div className={`${styles.line} ${styles[tone]}`}>
            <div className={styles.bar} aria-hidden="true">
                <div className={styles.fill} style={{width: `${pct}%`}}/>
            </div>
            <p className={styles.text}>{text}</p>
        </div>
    )
}