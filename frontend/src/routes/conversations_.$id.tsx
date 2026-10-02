import {createFileRoute, Link, redirect} from '@tanstack/react-router'
import {meQueryOptions} from "@/queries.ts";
import {conversationQueryOptions, useAcceptDM, useDeclineDM, useReopenDM} from "@/queries/conversation.ts";
import { ApiRequestError} from "@/lib/api.ts";
import {useConversationEvents} from "@/events/useConversationEvents.ts";
import {useSuspenseQuery} from "@tanstack/react-query";
import {Chat} from "@/components/Chat.tsx";
import {isoToLocalInput} from "@/utils/date.ts";


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

    return <>
        <div>
            <h1>{isOuting?`${conv.outing_title} --- ${conv.outing_starts_at&&isoToLocalInput(conv.outing_starts_at)}`:`${other?.name}`}</h1>
            {!isOuting && conv.dm_status === "pending" && conv.dm_initiator === me.id &&
                <p>Waiting for {other?.name} to accept.</p>}

            {!isOuting && conv.dm_status === "pending" && conv.dm_initiator !== me.id &&
                <div>
                    <p>{other?.name} wants to message you.</p>
                    <button onClick={() => acceptDM.mutate()}>Accept</button>
                    <button onClick={() => declineDM.mutate()}>Decline</button>
                </div>}
            {!isOuting && conv.dm_status === "declined" && conv.dm_declined_by === me.id &&
            <div>
                <p>This conversation is closed. Click Reopen button to start again</p>
                <button onClick={()=> reopenDM.mutate()}>Reopen</button>
                <Link className={'btn btn-quite'} to={"/inbox"} >Never mind</Link>
            </div>
            }
            {!isOuting && conv.dm_status === "declined" && conv.dm_declined_by !== me.id &&
            <p>This conversation is closed.</p>
            }
            {!isOuting && conv.dm_status == "accepted"&& <div>
                <p>Conversation is active</p>
                <button onClick={() => declineDM.mutate()}>Close conversation</button>
            </div>}
            <Chat cid={id} me={me} canPost={canPost} canModerate={canModerate} nameFor={ nameFor }></Chat>
        </div>
    </>

}
