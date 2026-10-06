import {createFileRoute, Link, redirect} from '@tanstack/react-router'
import {meQueryOptions} from "@/queries.ts";
import {conversationsQueryOptions} from "@/queries/conversation.ts";
import {useSuspenseQuery} from "@tanstack/react-query";

import type {ConversationSummary} from "@/types.ts";
import styles from "@/routes/inbox.module.css"
import {relativeTime} from "@/utils/date.ts";

export const Route = createFileRoute('/inbox')({
    beforeLoad: async ({context}) => {
        const me = await context.queryClient.ensureQueryData(meQueryOptions())
        if (!me) throw redirect({ to: '/login' })
    },
    loader : async ({context}) =>
        context.queryClient.ensureQueryData(conversationsQueryOptions()),
  component: InboxPage,
})

function InboxPage() {
    const {data:me} = useSuspenseQuery(meQueryOptions())
    const {data:conversations} = useSuspenseQuery(conversationsQueryOptions())
    const canAccept = (c:ConversationSummary) => c.kind ==="dm"&&c.dm_status==="pending"&&c.dm_initiator !== me?.id
    const requests = conversations.conversations.filter(c => canAccept(c))
    const dm = conversations.conversations.filter(c => c.kind === "dm" && !canAccept(c))
    const outing = conversations.conversations.filter(c => c.kind === "outing")


   if (!me) return
    return (
        <div className={styles.page}>
            <h1 className={styles.heading}>Inbox</h1>
            {requests.length > 0 && (
                <section className={styles.section}>
                    <h2 className={styles.sectionTitle}>Requests <span className={styles.count}>{requests.length}</span></h2>
                    <div className={styles.list}>{requests.map(r => <InboxRow key={r.id} cv={r} meId={me.id} />)}</div>
                </section>
            )}
            <section className={styles.section}>
                <h2 className={styles.sectionTitle}>Direct messages</h2>
                {dm.length === 0
                    ? <p className={styles.empty}>No direct messages yet. Message someone from an outing's roster.</p>
                    : <div className={styles.list}>{dm.map(r => <InboxRow key={r.id} cv={r} meId={me.id} />)}</div>}
            </section>
            <section className={styles.section}>
                <h2 className={styles.sectionTitle}>Outing chats</h2>
                {outing.length === 0
                    ? <p className={styles.empty}>Join or host an outing to get its group chat.</p>
                    : <div className={styles.list}>{outing.map(r => <InboxRow key={r.id} cv={r} meId={me.id} />)}</div>}
            </section>
        </div>
    )
}

function statusOf(cv: ConversationSummary, meId: string) {
    if (cv.kind !== 'dm') return null
    if (cv.dm_status === 'pending')
        return cv.dm_initiator === meId
            ? { text: 'Waiting', tone: styles.badgeMuted, preview: 'Request sent' }
            : { text: 'Request', tone: styles.badgeAccent, preview: 'Wants to message you' }
    if (cv.dm_status === 'declined')
        return { text: 'Closed', tone: styles.badgeMuted, preview: 'Conversation closed' }
    return null
}

function InboxRow({ cv, meId }: { cv: ConversationSummary; meId: string }) {
    const when = cv.last_message_at ?? cv.created_at
    const status = statusOf(cv, meId)
    const preview = cv.last_preview || status?.preview || 'No messages yet'

    return (
        <Link to="/conversations/$id" params={{ id: cv.id }} className={styles.row}>
            <span className={`${styles.avatar} ${cv.kind === 'outing' ? styles.avatarOuting : ''}`} aria-hidden>
                {cv.kind === 'outing' ? '⛰' : cv.title.charAt(0).toUpperCase()}
            </span>
            <span className={styles.body}>
                <span className={styles.top}>
                    <span className={styles.title}>{cv.title}</span>
                    {cv.unread_count > 0 && <span className={styles.unread}>{cv.unread_count}</span>}
                    {when && <time className={styles.time} dateTime={when}>{relativeTime(when)}</time>}
                </span>
                <span className={styles.bottom}>
                    <span className={`${styles.preview} ${cv.last_preview ? '' : styles.previewEmpty}`}>{preview}</span>
                    {status && <span className={`${styles.badge} ${status.tone}`}>{status.text}</span>}
                </span>
            </span>
        </Link>
    )
}