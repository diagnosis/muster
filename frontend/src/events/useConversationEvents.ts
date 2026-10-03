
import { useEvents} from "@/events/EventProvider.tsx";
import {useQueryClient} from "@tanstack/react-query";
import {useEffect} from "react";



export function useConversationEvents(cid:string){
    const { subscribe } = useEvents()
    const qc = useQueryClient()
    useEffect(()=>{
        const onPoke = (data: string) => {
            const poke = JSON.parse(data)
            if (poke.conversation_id === cid) qc.invalidateQueries({queryKey:['messages', cid]})
        }
        const onConvPoke = (data: string) => {
            const poke = JSON.parse(data)
            if (poke.conversation_id !== cid) return
            qc.invalidateQueries({ queryKey: ['conversation', cid] })
            qc.invalidateQueries({ queryKey: ['conversations'] })
        }
        const offs = [
            subscribe('message.created', onPoke),
            subscribe('message.deleted', onPoke),
            subscribe('dm.requested', onConvPoke),
            subscribe('dm.accepted', onConvPoke),
            subscribe('dm.declined', onConvPoke),
            subscribe('dm.reopened', onConvPoke),
        ]

        return () => offs.forEach(off => off())
    }, [cid,subscribe, qc])
}