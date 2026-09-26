import {createFileRoute, redirect} from '@tanstack/react-router'
import {meQueryOptions, outingQueryOptions} from "@/queries.ts";
import {useSuspenseQuery} from "@tanstack/react-query";
import {Chat} from "@/components/Chat.tsx";

export const Route = createFileRoute('/outings_/$id/conversation')({
  beforeLoad: async ({context, params}) => {
    const me = await context.queryClient.ensureQueryData(meQueryOptions())
    if (!me) throw redirect({ to: '/login' })

    const detail = await context.queryClient.ensureQueryData(outingQueryOptions(params.id))
    const isMember = detail.outing.host_id === me.id
    || detail.roster.some(r => r.hiker_id === me.id)
    if(!isMember) throw redirect({to: '/outings/$id', params: {id: params.id}})
  },
  loader: ({ context, params }) =>
      context.queryClient.ensureQueryData(outingQueryOptions(params.id)),
  component: ConversationComponent,
})

function ConversationComponent() {
  const {id} = Route.useParams()
  const { data: detail } = useSuspenseQuery(outingQueryOptions(id))
  const {data: me} = useSuspenseQuery(meQueryOptions())
  if (!me) return
  const cid = detail.outing.conversation_id
  if (!cid) return <p>Chat isn't available for this outing.</p>
  return <Chat cid={cid} detail={detail} me={me} />
}
