import { createFileRoute, redirect } from '@tanstack/react-router'
import {meQueryOptions} from "@/queries.ts";
import {conversationQueryOptions} from "@/queries/conversation.ts";
import { ApiRequestError} from "@/lib/api.ts";
import {useConversationEvents} from "@/events/useConversationEvents.ts";
import {useSuspenseQuery} from "@tanstack/react-query";
import {Chat} from "@/components/Chat.tsx";


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
    if (!me) return
    if (conv.kind == "outing"){

        return <Chat cid={id} oid={conv.outing_id} me={me} />
    }
    return <div>Hello "/conversations_/$id"!</div>
}
