import {createFileRoute, redirect} from '@tanstack/react-router'
import {meQueryOptions} from "@/queries.ts";
import {conversationsQueryOptions} from "@/queries/conversation.ts";
import {useSuspenseQuery} from "@tanstack/react-query";
import {useEffect} from "react";
import {useEvents} from "@/events/EventProvider.tsx";

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
    const {data:conversations} = useSuspenseQuery(conversationsQueryOptions())
    const { subscribe } = useEvents()
    useEffect(() => {
        return subscribe("message.created", (data:string)=>{

        })
    })
  return <div>Hello "/inbox"!</div>
}
