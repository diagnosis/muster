
import styles from '@/components/NotificationBell.module.css'
import {useNotifications} from "@/queries.ts";
import {useEffect, useRef, useState} from "react";
import type {NotificationEvent} from "@/types.ts";
import {relativeTime} from "@/utils/date.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import {useNavigate} from "@tanstack/react-router";
export function NotificationBell( ){
    const queryClient = useQueryClient()
    const navigate = useNavigate()
    const [open, setOpen] = useState(false)
    const ref = useRef<HTMLDivElement>(null)
    useEffect( () => {
        if (!open)return
        const onClick = (e: MouseEvent) => {
            if (ref.current && e.target instanceof Node && !ref.current.contains(e.target)){
                setOpen(false)
            }
        }
        document.addEventListener("mousedown", onClick)
        return () => document.removeEventListener("mousedown", onClick)
    }, [open])
    const { data } = useNotifications()
    const markAllReadMutation = useMutation({
        mutationFn: async () => {
            const res = await apiClient.post('/api/notifications/read-all')
            if (res.ok){
                return data
            }
            throw  new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            queryClient.invalidateQueries({queryKey: ['notifications']})
        }
    })
    const markRead = useMutation({
        mutationFn: async (id:string) =>{
            const res = await apiClient.post(`/api/notifications/${id}/read`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => queryClient.invalidateQueries({queryKey: ['notifications']})
    })

    const count = data?.unread_count ?? 0
    const notifications = data?.notifications ?? []
    return (
        <div className={styles.bellWrap} ref={ref}>
            <button
                className={`${styles.bellBtn} ${styles.bell}`}
                aria-label="Notifications"
                aria-expanded={open}
                onClick={() => setOpen(o => !o)}
            >
                🔔
                {count > 0 && <span className={styles.badge}>{count}</span>}
            </button>

            {open && (
                <div className={styles.notifDropdown}>
                    <div className={styles.notifHead}>
                        <span>Notifications</span>
                        {count > 0 && <button className={styles.markAll} disabled={markAllReadMutation.isPending} onClick={
                            () => markAllReadMutation.mutate()
                        } >Mark all read</button>}
                    </div>
                    {markAllReadMutation.isError&&<p className={"error"}>{markAllReadMutation.error.message}</p>}
                    <ul className={styles.notifList}>
                        {notifications.length === 0
                            ? <p className={styles.notifEmpty}>You're all caught up.</p>
                            : <ul className={styles.notifList}>
                                {notifications.map(n => (
                                    <li key={n.id} className={n.read_at ? styles.notifRow : `${styles.notifRow} ${styles.unread}`}
                                        onClick={() => {
                                            if (!n.read_at) markRead.mutate(n.id)
                                            const outingId = n.payload?.outing_id
                                            if (outingId) navigate({ to: '/outings/$id', params: { id: String(outingId) } })
                                            setOpen(false)   // close the dropdown after navigating
                                        }}
                                    >
                                        <span>{notificationText(n)}</span>
                                        <time className={styles.notifTime}>{relativeTime(n.created_at)}</time>
                                    </li>
                                ))}
                            </ul>
                        }
                    </ul>
                </div>
            )}
        </div>
    )
}

function notificationText(n: NotificationEvent): string {
    const title = n.payload?.outing_title ?? 'an outing'
    switch (n.kind) {
        case 'join_request_created':   return `New request to join ${title}`
        case 'join_request_approved':  return `Your request to join ${title} was approved`
        case 'join_request_declined':  return `Your request to join ${title} was declined`
        case 'join_request_withdrawn': return `Someone withdrew their request for ${title}`
        case 'member_removed':         return `You were removed from ${title}`
        case 'outing_cancelled':       return `${title} was cancelled`
        case 'outing_updated':         return `${title} was updated`
        default:                       return 'You have a new notification'
    }
}