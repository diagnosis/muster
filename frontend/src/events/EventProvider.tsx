import {createContext, type ReactNode, useCallback, useContext, useEffect, useRef} from "react";
import {refreshSession} from "@/lib/api.ts";
const API_BASE = import.meta.env.VITE_API_URL ?? ''
type Handler = (data:string) => void


const Ctx = createContext<{subscribe: (type: string, h:Handler) => () => void} | null >(null)

export function EventsProvider({children, enabled}:{children: ReactNode, enabled:boolean}){

    const handlers =useRef(new Map<string, Set<Handler>>())

    useEffect(() => {
        if(!enabled) return
        let es: EventSource | null = null
        let stopped = false

        const open = () => {
            es = new EventSource(`${API_BASE}/api/events`, {withCredentials: true})
            for (const type of ["message.created", "message.deleted"]){
                es.addEventListener(type, (e)=> {
                    handlers.current.get(type)?.forEach(h=>h((e as MessageEvent).data))
                })
            }
            es.onerror = async () => {
                es?.close()
                if (stopped) return
                const ok = await refreshSession()
                if (ok) open(); else stopped = true
            }
        }
        open()
        return () => {stopped= true; es?.close()}
    }, [enabled])

    const subscribe = useCallback((type: string, h: Handler) => {
        const set = handlers.current.get(type) ?? new Set()
        set.add(h); handlers.current.set(type, set)
        return () => { set.delete(h) }
    }, [])
    return <Ctx.Provider value={{ subscribe }}>{children}</Ctx.Provider>
}
export function useEvents() {
    const v = useContext(Ctx)
    if (!v) throw new Error("useEvents outside EventsProvider")
    return v
}