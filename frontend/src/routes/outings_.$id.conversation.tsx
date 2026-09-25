import {createFileRoute, redirect} from '@tanstack/react-router'
import {meQueryOptions, outingQueryOptions} from "@/queries.ts";
import {useSuspenseQuery} from "@tanstack/react-query";
import {Chat} from "@/components/Chat.tsx";

export const Route = createFileRoute('/outings_/$id/conversation')({
  beforeLoad: async ({context}) => {
    const me = await context.queryClient.ensureQueryData(meQueryOptions())
    if (!me) throw redirect({ to: '/login' })
  },
  loader: ({ context, params }) =>
      context.queryClient.ensureQueryData(outingQueryOptions(params.id)),
  component: ConversationComponent,
})

function ConversationComponent() {
  const {id} = Route.useParams()
  const { data: detail } = useSuspenseQuery(outingQueryOptions(id))
  const cid = detail.outing.conversation_id
  if (!cid) return <p>Chat isn't available for this outing.</p>
  return <Chat cid={cid} outing={detail.outing} />
}
