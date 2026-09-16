import {expect, test} from '@playwright/test'
import {asUser, BASE} from "../fixtures";
import {
    accept,
    addComment,
    cancelOuting,
    createOuting,
    deleteComment,
    likeComment,
    listComments,
    requestJoin, unlikeComment
} from "../api";
import {unwrap} from "../envelope";
import {Comment, JoinRequestResponse, ListCommentResponse, OutingResponse} from "../types";

test.describe("comments and likes", ()=>{
    test(`add comment -> add likes -> validate like counts and like by me `, async ()=>{
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
        await unwrap(likeComment(host.ctx, outing.id, commentRes3.id), 200)
        await unwrap(likeComment(requested.ctx, outing.id, commentRes2.id), 200)
        const {comments} = await unwrap<ListCommentResponse>(listComments(accepted.ctx, outing.id), 200)
        for (const comment of comments){
                if (comment.hiker_id == host.id){
                    expect(comment.like_count).toBe(2)
                    expect(comment.liked_by_me).toBe(true)
                    expect(comment.author_name).toBe(host.user.name)
                }
                if (comment.hiker_id == accepted.id){
                    expect(comment.like_count).toBe(1)
                    expect(comment.liked_by_me).toBe(false)
                    expect(comment.author_name).toBe(accepted.user.name)
                }
        }
    });

    test('reply to a reply -> 409', async ()=>{
        const host = await asUser(BASE)
        const requested = await asUser(BASE)
        const accepted = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        await unwrap<JoinRequestResponse>(requestJoin(requested.ctx, outing.id), 201)
        const r2 = await unwrap<JoinRequestResponse>(requestJoin(accepted.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r2.id), 200)

        const comment1 = await unwrap<Comment>(addComment(host.ctx, outing.id, {body:"please bring bear spray", parent_id:null}), 201)
        const comment2 = await unwrap<Comment>(addComment(accepted.ctx, outing.id, {body:"i dont have one do u want me to buy?", parent_id:comment1.id}), 201)
        await unwrap<Comment>(addComment(requested.ctx, outing.id, {body:"dont buy i have one",parent_id:comment2.id}), 409)
    });

    test('a stranger neither comments nor list -> 403', async ()=> {
        const host = await asUser(BASE)
        const hiker1 = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const stranger = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const r1 = await unwrap<JoinRequestResponse>(requestJoin(hiker1.ctx, outing.id), 201)
        const r2 = await unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r1.id), 200)
        await unwrap(accept(host.ctx, r2.id), 200)

        const c1 = await unwrap<Comment>(addComment(host.ctx, outing.id, {body:"So cold tomorrow", parent_id:null}), 201)
        await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"yeah we need to bring thicker clothes", parent_id:c1.id}), 201)
        await unwrap<Comment>(addComment(hiker2.ctx, outing.id, {body:"better to carry emergency clothing", parent_id:c1.id}), 201)
        await unwrap(listComments(stranger.ctx, outing.id), 403)
        await unwrap(addComment(stranger.ctx, outing.id, {body:"providing website development for please call us!", parent_id:null}), 403)
    });

    test("cancelled outing read-only -> comment -> 409", async ()=> {
        const host = await asUser(BASE)
        const hiker1 = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const r1 = await unwrap<JoinRequestResponse>(requestJoin(hiker1.ctx, outing.id), 201)
        await unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r1.id), 200)
        await unwrap(cancelOuting(host.ctx, outing.id), 200)
        await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"Why cancelled?", parent_id:null}), 409)
    });

    test("owner deletes own comment; list shows empty body, replies survive", async ()=>{
        const host = await asUser(BASE)
        const hiker1 = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const r1 = await unwrap<JoinRequestResponse>(requestJoin(hiker1.ctx, outing.id), 201)
        const r2 = await unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r1.id), 200)
        await unwrap(accept(host.ctx, r2.id), 200)

        const c1 = await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"So cold tomorrow", parent_id:null}), 201)
        const c2 = await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"we need to bring thicker clothes", parent_id:c1.id}), 201)
        const c3 = await unwrap<Comment>(addComment(hiker2.ctx, outing.id, {body:"better to carry emergency clothing", parent_id:c1.id}), 201)
        await unwrap(deleteComment(hiker1.ctx, outing.id, c1.id), 200)
        const commentList = await unwrap<ListCommentResponse>(listComments(host.ctx, outing.id), 200)
        const parent = commentList.comments.find(c => c.id === c1.id)
        expect(parent?.deleted).toBe(true)
        expect(parent?.body).toBe('')
        const child1 = commentList.comments.find(c => c.id === c2.id)
        expect(child1?.deleted).toBe(false)
        expect(child1?.body).toBe(c2.body)
        const child2 = commentList.comments.find(c => c.id === c3.id)
        expect(child2?.deleted).toBe(false)
        expect(child2?.body).toBe(c3.body)
    })

    test("host deletes other's comment; list shows empty body and deleted true, replies survive", async () => {
        const host = await asUser(BASE)
        const hiker1 = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const r1 = await unwrap<JoinRequestResponse>(requestJoin(hiker1.ctx, outing.id), 201)
        const r2 = await unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r1.id), 200)
        await unwrap(accept(host.ctx, r2.id), 200)

        const c1 = await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"My car broken", parent_id:null}), 201)
        const c2 = await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"Can someone else pick me up?", parent_id:c1.id}), 201)
        const c3 = await unwrap<Comment>(addComment(hiker2.ctx, outing.id, {body:"sure i can", parent_id:c1.id}), 201)
        await unwrap(deleteComment(host.ctx, outing.id, c1.id), 200)
        const commentList = await unwrap<ListCommentResponse>(listComments(host.ctx, outing.id), 200)
        const parent = commentList.comments.find(c => c.id === c1.id)
        expect(parent?.deleted).toBe(true)
        expect(parent?.body).toBe('')
        const child1 = commentList.comments.find(c => c.id === c2.id)
        expect(child1?.deleted).toBe(false)
        expect(child1?.body).toBe(c2.body)
        const child2 = commentList.comments.find(c => c.id === c3.id)
        expect(child2?.deleted).toBe(false)
        expect(child2?.body).toBe(c3.body)
    });

    test("neither stranger nor other members cannot delete comment -> 403", async()=> {
        const host = await asUser(BASE)
        const hiker1 = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const stranger = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const r1 = await unwrap<JoinRequestResponse>(requestJoin(hiker1.ctx, outing.id), 201)
        const r2 = await unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r1.id), 200)
        await unwrap(accept(host.ctx, r2.id), 200)

        const c1 = await unwrap<Comment>(addComment(host.ctx, outing.id, {body:"So cold tomorrow", parent_id:null}), 201)
        const c2 = await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"yeah we need to bring thicker clothes", parent_id:c1.id}), 201)
        const c3 = await unwrap<Comment>(addComment(hiker2.ctx, outing.id, {body:"better to carry emergency clothing", parent_id:c1.id}), 201)
        await unwrap(deleteComment(stranger.ctx, outing.id, c2.id), 403)
        await unwrap(deleteComment(hiker1.ctx, outing.id, c3.id), 403)
    });

    test("like is idempotent, unlike removes; like -> list count 1; like again -> 200, count still 1. unlike is same as well", async () => {
        const host = await asUser(BASE)
        const hiker1 = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const r1 = await unwrap<JoinRequestResponse>(requestJoin(hiker1.ctx, outing.id), 201)
        await unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r1.id), 200)
        const c2 = await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"I love hiking", parent_id:null}), 201)
        await unwrap<Comment>(addComment(hiker2.ctx, outing.id, {body:"me too", parent_id:c2.id}), 201)
        await unwrap(likeComment(host.ctx, outing.id, c2.id), 200)
        await unwrap(likeComment(hiker2.ctx, outing.id, c2.id), 200)
        let commentList = await unwrap<ListCommentResponse>(listComments(host.ctx, outing.id), 200)
        let c2Comment  = commentList.comments.find(c => c.id === c2.id)
        expect(c2Comment?.like_count).toBe(2)
        expect(c2Comment?.liked_by_me).toBe(true)
        await unwrap(likeComment(host.ctx, outing.id, c2.id), 200)
        commentList = await unwrap<ListCommentResponse>(listComments(host.ctx, outing.id), 200)
        c2Comment  = commentList.comments.find(c => c.id === c2.id)
        expect(c2Comment?.like_count).toBe(2)
        expect(c2Comment?.liked_by_me).toBe(true)

        await unwrap(unlikeComment(host.ctx, outing.id, c2.id), 200)
        commentList = await unwrap<ListCommentResponse>(listComments(host.ctx, outing.id), 200)
        c2Comment  = commentList.comments.find(c => c.id === c2.id)
        expect(c2Comment?.like_count).toBe(1)
        expect(c2Comment?.liked_by_me).toBe(false)

        await unwrap(unlikeComment(host.ctx, outing.id, c2.id), 200)
        commentList = await unwrap<ListCommentResponse>(listComments(host.ctx, outing.id), 200)
        c2Comment  = commentList.comments.find(c => c.id === c2.id)
        expect(c2Comment?.like_count).toBe(1)
        expect(c2Comment?.liked_by_me).toBe(false)
    });

    test("cannot like deleted comment -> 409", async ()=>{
        const host = await asUser(BASE)
        const hiker1 = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const r1 = await unwrap<JoinRequestResponse>(requestJoin(hiker1.ctx, outing.id), 201)
        const r2 = await unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, r1.id), 200)
        const c2 = await unwrap<Comment>(addComment(hiker1.ctx, outing.id, {body:"I love hiking", parent_id:null}), 201)
        await unwrap(deleteComment(hiker1.ctx, outing.id, c2.id), 200)
        await unwrap(likeComment(host.ctx, outing.id, c2.id), 409)
    })

})