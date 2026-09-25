
import {useEvents} from "@/events/EventProvider.tsx";
import {useQueryClient} from "@tanstack/react-query";
import {useEffect} from "react";



export function useConversationEvents(cid:string){
    const { subscribe } = useEvents()
    const qc = useQueryClient()
    useEffect(()=>{
        const onPoke = (data: string) => {
            const poke = JSON.parse(data)
            if (poke.conversation_id == cid) qc.invalidateQueries({queryKey:['messages', cid]})
        }
        const off1 = subscribe('message.created', onPoke)
        const off2 = subscribe('message.deleted', onPoke)
        return () => {off1(); off2()}
    }, [cid,subscribe, qc])
}