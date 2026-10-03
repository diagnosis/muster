import {createContext, type ReactNode, useCallback, useContext, useEffect, useRef} from "react";
import {refreshSession} from "@/lib/api.ts";
import {useQueryClient} from "@tanstack/react-query";
const API_BASE = import.meta.env.VITE_API_URL ?? ''
type Handler = (data:string) => void


export const EVENT_TYPES =["message.created", "message.deleted", "notification.created", "dm.requested", "dm.accepted", "dm.declined", "dm.reopened"]

const Ctx = createContext<{subscribe: (type: string, h:Handler) => () => void} | null >(null)

export function EventsProvider({children, enabled}:{children: ReactNode, enabled:boolean}){
    const qc = useQueryClient()
    const handlers =useRef(new Map<string, Set<Handler>>())

    useEffect(() => {
        if(!enabled) return
        let es: EventSource | null = null
        let stopped = false
        let delay = 1000
        let opened = false
        let timer: ReturnType<typeof setTimeout> | null = null

        const reconnect = async () => {
            es?.close()
            if (stopped) return
            const status = await refreshSession()
            if (stopped) return                       // cleanup ran while we awaited
            if (status === 401) { stopped = true; return }
            if (status === 200) { open(); return }
            timer = setTimeout(reconnect, delay)      // server down: try again later
            delay = Math.min(delay * 2, 30_000)
        }

        const open = () => {
            es = new EventSource(`${API_BASE}/api/events`, {withCredentials: true})
            for (const type of EVENT_TYPES){
                es.addEventListener(type, (e)=> {
                    handlers.current.get(type)?.forEach(h=>h((e as MessageEvent).data))
                })
            }
            es.onopen = () => {
                delay = 1000
                if (opened) qc.invalidateQueries()
                opened = true
            }
            es.onerror = reconnect
        }
        open()
        return () => {stopped= true;if (timer) clearTimeout(timer); es?.close()}
    }, [enabled,qc])

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

