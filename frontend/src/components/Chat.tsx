// src/components/Chat.tsx

import {useDeleteMessage, useListMessages, usePostMessage} from "@/queries/message.ts";
import type {MeResponse} from "@/types.ts";
import {useState} from "react";
import styles from '@/components/Chat.module.css'
import {TrashIcon} from "@/components/Icons.tsx";
import {messageTime} from "@/utils/date.ts";
import {useMarkReadWhenSeen} from "@/events/useMarkReadWhenSeen.ts";

interface ChatProps {
    cid: string
    me: MeResponse
    canPost: boolean
    canModerate: boolean
    nameFor : (id: string) => string
    showAuthors?: boolean
    disabledHint?: string
}

export function Chat({cid, me, canPost,canModerate, nameFor, showAuthors, disabledHint}:ChatProps){

    const [body, setBody] = useState("")
    const [selected, setSelected] = useState<string|null>(null)
    const messages = useListMessages(cid)
    const postMessage = usePostMessage(cid)
    const deleteMessage = useDeleteMessage(cid)

    const list = messages.data?.messages ?? []
    useMarkReadWhenSeen(cid, list.at(-1)?.seq)
    function send() {
        if (!canPost || !body.trim() || postMessage.isPending) return
        postMessage.mutate({body}, {onSuccess: () => setBody("")})
    }

    return (
        <div className={styles.chat}>
            <div className={styles.messages}>
                {list.length === 0 && <p className={styles.empty}>No messages yet.</p>}
                {list.map(m => {
                    const mine = m.hiker_id === me.id
                    const canDelete = mine || canModerate
                    const isSelected = selected === m.id
                    return (
                        <div key={m.id} className={`${styles.message} ${mine ? styles.mine : styles.theirs}`}>
                            {!mine && showAuthors && <span className={styles.author}>{nameFor(m.hiker_id)}</span>}
                            <div className={styles.row}>
                                {canDelete && isSelected &&
                                    <button className={styles.deleteBtn}
                                            onClick={() => deleteMessage.mutate(m.id, {onSuccess: () => setSelected(null)})}
                                            disabled={deleteMessage.isPending}
                                            aria-label={`delete message ${m.id}`}
                                    ><TrashIcon/></button>
                                }
                                {canDelete
                                    ? <button type="button"
                                              className={`${styles.bubble} ${styles.bubbleBtn} ${isSelected ? styles.bubbleSelected : ''}`}
                                              aria-expanded={isSelected}
                                              onClick={() => setSelected(isSelected ? null : m.id)}
                                    >{m.body}</button>
                                    : <p className={styles.bubble}>{m.body}</p>}
                            </div>
                            <time className={styles.time} dateTime={m.created_at}>{messageTime(m.created_at)}</time>
                        </div>
                    )
                })}
            </div>
            <form className={`${styles.composer} ${canPost ? '' : styles.composerDisabled}`}
                  onSubmit={(e) => { e.preventDefault(); send() }}>
                <textarea
                    className={styles.input}
                    rows={1}
                    aria-label='chat-box'
                    enterKeyHint="send"
                    disabled={!canPost}
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                    onKeyDown={(e) => {
                        if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                            e.preventDefault()
                            send()
                        }
                    }}
                    placeholder={canPost ? "Write a message" : (disabledHint ?? "You can't send messages here")}
                />
                <button className={`btn-primary ${styles.send}`} type='submit' disabled={!canPost}>Send</button>
            </form>
        </div>
    )
}