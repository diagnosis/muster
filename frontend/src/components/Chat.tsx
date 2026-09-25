// src/components/Chat.tsx

import { useListMessages, usePostMessage} from "@/queries/message.ts";
import type {Outing} from "@/types.ts";
import {useState} from "react";
import {useConversationEvents} from "@/events/useConversationEvents.ts";
interface ChatProps {
    cid: string
    outing: Outing
}

export function Chat({cid, outing}:ChatProps){
    useConversationEvents(cid)
    const [body, setBody] = useState("")
    const messages = useListMessages(cid)
    const postMessage = usePostMessage(cid)

    return <>
        <div>
            <h1>{outing.title}</h1>
            <div>
                {messages.data?.messages.map(m => <p key={m.id}>
                        {m.body}
                    </p>
                )}
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