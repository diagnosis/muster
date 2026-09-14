import {expect, test} from '@playwright/test'
import {asUser, BASE} from "../fixtures";
import {accept, addComment, createOuting, likeComment, listComments, requestJoin} from "../api";
import {unwrap} from "../envelope";
import {Comment, JoinRequestResponse, ListCommentResponse, OutingResponse} from "../types";

test.describe("comments and likes", ()=>{
    test(`add comment`, async ()=>{
        const host = await asUser(BASE)
        const requested = await asUser(BASE)
        const accepted = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        await unwrap<JoinRequestResponse>(requestJoin(requested.ctx, outing.id), 201)
        const r2 = await unwrap<JoinRequestResponse>(requestJoin(accepted.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r2.id), 200)

        const commentRes1 = await unwrap<Comment>(addComment(requested.ctx, outing.id, {body:"hello", parent_id:null}), 201)
        const commentRes2 = await unwrap<Comment>(addComment(accepted.ctx, outing.id, {body:"hi there", parent_id:commentRes1.id}), 201)
        const commentRes3 = await unwrap<Comment>(addComment(host.ctx, outing.id, {body:"if weather queality will have not been improved by tomorrow. I will cancel", parent_id:null}), 201)
        expect(commentRes2.parent_id).toBe(commentRes1.id)
        expect(commentRes1.body).toBe("hello")
        expect(commentRes2.body).toBe("hi there")

        await unwrap(likeComment(accepted.ctx, outing.id, commentRes3.id), 200)
        const {comments} = await unwrap<ListCommentResponse>(listComments(accepted.ctx, outing.id), 200)
        for (const comment of comments){
                if (comment.hiker_id == host.id){
                    expect(comment.like_count).toBe(1)
                    expect(comment.liked_by_me).toBe(true)
                }
        }
    })
})