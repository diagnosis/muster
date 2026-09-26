// src/components/Chat.tsx

import {useDeleteMessage, useListMessages, usePostMessage} from "@/queries/message.ts";
import type {Detail, MeResponse} from "@/types.ts";
import {useState} from "react";
import {useConversationEvents} from "@/events/useConversationEvents.ts";
import styles from '@/components/Chat.module.css'
interface ChatProps {
    cid: string
    detail: Detail
    me: MeResponse
}

export function Chat({cid, detail, me}:ChatProps){
    useConversationEvents(cid)
    const [body, setBody] = useState("")
    const messages = useListMessages(cid)
    const postMessage = usePostMessage(cid)
    const deleteMessage = useDeleteMessage(cid)
    return <>
        <div>
            <h1>{detail.outing.title}</h1>
            <div>
                {messages.data?.messages.map(m =>
                   <div key={m.id} className={styles.messageRow}>
                       <p >
                           {m.hiker_id === detail.host.hiker_id?detail.host.name:(detail.roster.find(r => r.hiker_id === m.hiker_id))?.name}
                           {` : ${m.body}`}
                       </p>
                       {(me.id === m.hiker_id || me.id === detail.host.hiker_id) &&
                           <button onClick={(e)=>{
                               e.preventDefault()
                               deleteMessage.mutate(m.id)
                           }}
                                   disabled={deleteMessage.isPending}
                                   aria-label={`delete message ${m.id}`}
                           >delete</button>
                       }
                   </div>
                )
                }

            </div>
            <form onSubmit={(e)=>{
                e.preventDefault()
                if (!body.trim()) return
                postMessage.mutate({body}, {onSuccess: ()=> setBody("")})
            }}>
                <textarea
                    aria-label='chat-box'
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                    placeholder={"start typing..."}
                >
                </textarea>
                <button type='submit'>send</button>
            </form>
        </div>
    </>
}