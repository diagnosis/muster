
import styles from '@/components/Header.module.css'
import {useNotifications} from "@/queries.ts";
export function NotificationBell(){
    const { data } = useNotifications()
    const count = data?.unread_count ?? 0
    return  (<button className={`${styles.bellBtn} ${styles.bell}`} aria-label="Notifications">
        🔔
        {count > 0 && <span className={styles.badge}>{count}</span>}
    </button>)
}