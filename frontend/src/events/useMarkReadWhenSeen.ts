import {useMarkRead} from "@/queries/message.ts";
import {useEffect, useRef} from "react";


export function useMarkReadWhenSeen(cid:string, lastSeq:number|undefined){
    const {mutate} = useMarkRead(cid)
    const reported = useRef(0)

    useEffect(()=>{reported.current = 0}, [cid])
    useEffect(()=>{
        const report = () => {
            if (lastSeq == undefined)return
            if (document.visibilityState !== 'visible')return
            if (lastSeq <= reported.current) return
            reported.current = lastSeq
            mutate({seq:lastSeq})
        }
        report()
        document.addEventListener('visibilitychange', report)
        return () => document.removeEventListener('visibilitychange', report)
    }, [lastSeq, mutate])
}