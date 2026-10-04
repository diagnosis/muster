import {createFileRoute, Link, redirect} from '@tanstack/react-router'
import {meQueryOptions} from "@/queries.ts";
import {conversationQueryOptions, useAcceptDM, useDeclineDM, useReopenDM} from "@/queries/conversation.ts";
import { ApiRequestError} from "@/lib/api.ts";
import {useConversationEvents} from "@/events/useConversationEvents.ts";
import {useSuspenseQuery} from "@tanstack/react-query";
import {Chat} from "@/components/Chat.tsx";
import styles from "@/routes/conversation.module.css"
import {ChevronLeftIcon, MountainIcon} from "@/components/Icons.tsx";
import {outingWhen} from "@/utils/date.ts";

export const Route = createFileRoute('/conversations_/$id')({
    beforeLoad: async ({ context }) => {
        const me = await context.queryClient.ensureQueryData(meQueryOptions())
        if (!me) throw redirect({ to: '/login' })
    },
    loader: ({ context, params }) =>
        context.queryClient.ensureQueryData(conversationQueryOptions(params.id)),
    errorComponent: ({ error }) => (
        <p>{error instanceof ApiRequestError && error.httpStatus === 403
            ? "You're not part of this conversation."
            : "Couldn't load this conversation."}</p>
    ),
    component: ConversationPage,
})

function ConversationPage() {
    const {id} = Route.useParams()
    useConversationEvents(id)
    const {data: conv} = useSuspenseQuery(conversationQueryOptions(id))
    const {data: me} = useSuspenseQuery(meQueryOptions())
    const acceptDM = useAcceptDM(id)
    const declineDM = useDeclineDM(id)
    const reopenDM = useReopenDM(id)
    if (!me) return
    const isOuting = conv.kind === "outing"
    const canModerate = isOuting && conv.outing_host_id === me.id
    let canPost = isOuting
    if (!isOuting) {
        canPost = conv.dm_status === "accepted"
            || (conv.dm_status === "pending" && conv.dm_initiator === me.id)
    }
    const other = conv.participants.find(p => p.hiker_id !== me.id)
    const nameFor = (hikerID: string) =>
        conv.participants.find(p => p.hiker_id === hikerID)?.name ?? "someone"
    const otherName = other?.name || "Someone"
    const pending = !isOuting && conv.dm_status === "pending"
    const declined = !isOuting && conv.dm_status === "declined"
    const accepted = !isOuting && conv.dm_status === "accepted"
    const iStarted = conv.dm_initiator === me.id
    const iClosed = conv.dm_declined_by === me.id

    let disabledHint: string | undefined
    if (pending && !iStarted) disabledHint = "Accept the request to reply"
    if (declined) disabledHint = "This conversation is closed"

    return (
        <div className={styles.page}>
            <Link to="/inbox" className={styles.back}><ChevronLeftIcon/> Inbox</Link>

            <header className={styles.head}>
            <span className={`${styles.avatar} ${isOuting ? styles.avatarOuting : ''}`} aria-hidden="true">
                {isOuting ? <MountainIcon/> : otherName.charAt(0).toUpperCase()}
            </span>
                <div className={styles.headText}>
                    <h1 className={styles.title}>{isOuting ? conv.outing_title : otherName}</h1>
                    <p className={styles.sub}>
                        {isOuting
                            ? (conv.outing_starts_at ? outingWhen(conv.outing_starts_at) : "Outing chat")
                            : (pending && !iStarted ? "Message request" : "Direct message")}
                    </p>
                </div>
                {accepted &&
                    <button className={styles.closeBtn} onClick={() => declineDM.mutate()}>Close conversation</button>}
            </header>

            {pending && iStarted &&
                <section className={styles.notice}>
                    <div>
                        <h2 className={styles.noticeTitle}>Waiting for {otherName} to accept</h2>
                        <p className={styles.noticeText}>You can send one message now. The rest opens up once they accept.</p>
                    </div>
                </section>}

            {pending && !iStarted &&
                <section className={styles.notice}>
                    <div>
                        <h2 className={styles.noticeTitle}>{otherName} wants to message you</h2>
                        <p className={styles.noticeText}>Accept to reply. If you decline, the request closes and they are not notified.</p>
                    </div>
                    <div className={styles.noticeActions}>
                        <button className="btn-primary" onClick={() => acceptDM.mutate()}>Accept</button>
                        <button onClick={() => declineDM.mutate()}>Decline</button>
                    </div>
                </section>}

            {declined && iClosed &&
                <section className={`${styles.notice} ${styles.noticeMuted}`}>
                    <div>
                        <h2 className={styles.noticeTitle}>You closed this conversation</h2>
                        <p className={styles.noticeText}>Nobody can send messages until you reopen it.</p>
                    </div>
                    <div className={styles.noticeActions}>
                        <button className="btn-primary" onClick={() => reopenDM.mutate()}>Reopen</button>
                        <Link className={styles.noticeLink} to="/inbox">Back to inbox</Link>
                    </div>
                </section>}

            {declined && !iClosed &&
                <section className={`${styles.notice} ${styles.noticeMuted}`}>
                    <div>
                        <h2 className={styles.noticeTitle}>{otherName} closed this conversation</h2>
                        <p className={styles.noticeText}>You can still read it. Only they can reopen it.</p>
                    </div>
                </section>}

            <Chat cid={id} me={me} canPost={canPost} canModerate={canModerate} nameFor={nameFor}
                  showAuthors={isOuting} disabledHint={disabledHint}/>
        </div>
    )

}
