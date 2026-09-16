// src/components/Comments.tsx

import {useAddComment, useDeleteComment, useLikeComment, useListComments, useUnlikeComment} from "@/queries/comment.ts";
import {relativeTime} from "@/utils/date.ts";
import type {CommentView} from "@/types.ts";
import styles from '@/components/Comments.module.css'
import {useState} from "react";
import {useMeQuery} from "@/queries.ts";
import {Modal} from "@/components/Modal.tsx";
import {COMMENT_REMOVED} from "@/lib/copy.ts";

interface CommentsProp{
    outingId: string
    hostId: string
    readOnly: boolean
}
export function Comments({outingId, hostId, readOnly}:CommentsProp){
    const { data, error, isError, isPending } = useListComments(outingId)
    const [replyingTo, setReplyingTo] = useState<string | null>(null)
    if (isPending) return <p>Loading comments…</p>
    if (isError) return <p>{error.message}</p>
    const comments = data.comments
    const topLevel = comments.filter(c => c.parent_id == null)
    const repliesOf = (id: string) => comments.filter(c => c.parent_id === id)
    return (
        <section className={styles.section}>
            <h2 className={"subheading"}>Discussion</h2>
            <div className={styles.thread}>
                {topLevel.length===0&&
                    <p className={styles.meta}>No comments yet - start the conversation.</p>}
            {topLevel.length >0 && topLevel.map(c => (
                <div key={c.id}>
                    <CommentRow c={c} outingId={outingId} hostId={hostId} onReply={() => setReplyingTo(c.id)} />
                    {replyingTo === c.id && (
                        <CommentForm outingID={outingId} parentID={c.id} onDone={() => setReplyingTo(null)} />
                    )}
                    <div className={styles.replies}>
                        {repliesOf(c.id).map(rc => <CommentRow key={rc.id} c={rc} outingId={outingId} hostId={hostId}/>)}
                    </div>
                </div>
            ))}
            </div>
            {!readOnly && <CommentForm outingID={outingId} parentID={null}/>}
        </section>
    )
}

function CommentRow({ c, outingId, hostId, onReply }: { c: CommentView; outingId: string, hostId:string, onReply?: ()=>void}) {
    const [currentComment, setCurrentComment] = useState<CommentView|null>(null)
    const {data:me} = useMeQuery()
    const like = useLikeComment(outingId)
    const unlike = useUnlikeComment(outingId)
    const toggle = () => c.liked_by_me ? unlike.mutate(c.id) : like.mutate(c.id)
    const busy = like.isPending || unlike.isPending
    const likeErr = like.error || unlike.error
    const isError = like.isError || unlike.isError

    const del = useDeleteComment(outingId)

    const canDelete = me?.id === c.hiker_id || me?.id === hostId
    return (
        <article className={styles.row}>
            <div className={styles.meta}>
                <span className={styles.author}>{c.author_name}</span> · <time>{relativeTime(c.created_at)}</time>
            </div>
            <p className={c.deleted ? `${styles.body} ${styles.stub}` : styles.body}>
                {c.deleted ? COMMENT_REMOVED : c.body}
            </p>
            {!c.deleted && (
                <div className={styles.actions}>
                    <button aria-label={"Like"}
                        className={`${styles.actionBtn} ${c.liked_by_me ? styles.liked : ''}`} aria-pressed={c.liked_by_me} disabled={busy} onClick={toggle}>
                        ♥ {c.like_count}
                    </button>
                    {onReply && <button type="button" className={styles.actionBtn} onClick={onReply}>Reply</button>}
                    {isError && <span className={styles.meta}>{likeErr?.message}</span>}
                    {canDelete&&<button className={styles.actionBtn} onClick={()=>{setCurrentComment(c)}}>Delete</button>}
                </div>
            )}
            {currentComment&&<Modal title={'Delete this comment? Replies will stay.'} onClose={()=>setCurrentComment(null)}>
                <div>
                    <div className={styles.actions}>
                        <button className="btn btn-danger" disabled={del.isPending} onClick={() => del.mutate(currentComment.id, {
                            onSuccess: () => setCurrentComment(null)
                        })}>Yes</button>
                        <button className={"btn btn-quite"} onClick={() => setCurrentComment(null)}>Never mind</button>
                    </div>
                    {del.isError&&<p className={"error"}>{del.error.message}</p>}
                </div>
            </Modal>}
        </article>
    )
}

function CommentForm({outingID,parentID, onDone}:{outingID:string, parentID:string|null, onDone?: ()=>void}){
    const [body, setBody] = useState("")
    const add = useAddComment(outingID)
    return(
        <form className={styles.form} onSubmit={(e) =>{
            e.preventDefault();
            add.mutate({body, parent_id:parentID}, {onSuccess: () => {setBody(''); onDone?.()}})
        }}>
            <textarea
                className={styles.textarea}
                value={body}
                placeholder={`${parentID?'Write a reply...':'Add to the discussion'}`}
                onChange={e=>setBody(e.target.value)}
            >
            </textarea>
            <div className={styles.formRow}>
                <button className={"btn btn-primary"} type='submit' disabled={add.isPending||!body.trim()}>{`${parentID?'Post reply':'Post'}`}</button>
            </div>
            {add.isError && <p className={"error"}>{add.error.message}</p>}
        </form>
    )
}