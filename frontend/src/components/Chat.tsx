// src/components/Chat.tsx

import {useDeleteMessage, useListMessages, usePostMessage} from "@/queries/message.ts";
import type {MeResponse} from "@/types.ts";
import {useState} from "react";
import styles from '@/components/Chat.module.css'

interface ChatProps {
    cid: string
    me: MeResponse
    canPost: boolean
    canModerate: boolean
    nameFor : (id: string) => string
}

export function Chat({cid, me, canPost,canModerate, nameFor}:ChatProps){

    const [body, setBody] = useState("")
    const messages = useListMessages(cid)
    const postMessage = usePostMessage(cid)
    const deleteMessage = useDeleteMessage(cid)

    return <>
        <div>
            <div>
                {messages.data?.messages.map(m =>
                   <div key={m.id} className={styles.messageRow}>
                       <p >
                           <strong>{nameFor(m.hiker_id)}</strong>
                           {` : ${m.body}`}
                       </p>
                       {(me.id === m.hiker_id || canModerate) &&
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
                    disabled={!canPost}
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                    placeholder={"start typing..."}
                >
                </textarea>
                <button type='submit' disabled={!canPost}>send</button>
            </form>
        </div>
    </>
}