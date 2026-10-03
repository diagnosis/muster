// backend/e2e/tests/dm.spec.ts


import {APIRequestContext, expect, test, } from "@playwright/test";
import {asUser, BASE} from "../fixtures";
import {unwrap, unwrapError, unwrapExpectNoContent} from "../envelope";
import {
    ApiErrorBody,
    ConversationResponse,
    JoinRequestResponse, ListConversationsResponse,
    Message,
    OutingResponse,
    RegisterRequest
} from "../types";
import {
    accept,
    acceptDM,
    createOuting,
    declineDM,
    listConversations,
    postMessage,
    reopenDM,
    requestJoin,
    startDM
} from "../api";


test.describe("dm specs", () => {
    let host: { ctx: APIRequestContext; id: string; user?: RegisterRequest; },
        member1: { ctx: APIRequestContext; id: string; user?: RegisterRequest; },
        member2: { ctx: APIRequestContext; id: string; user?: RegisterRequest; },
        pending: { ctx: APIRequestContext; id: string; user?: RegisterRequest; },
        stranger: { id: string; ctx: APIRequestContext; user?: RegisterRequest; }, outing: OutingResponse,
        jr1: JoinRequestResponse, jr2: JoinRequestResponse, jr3

    test.beforeEach(async()=> {
        host = await asUser(BASE)
        member1 = await asUser(BASE)
        member2 = await asUser(BASE)
        pending = await asUser(BASE)
        stranger = await asUser(BASE)

        outing = await unwrap<OutingResponse>( createOuting(host.ctx, {max_size:4, host_seats:4}), 201)
        jr1 = await unwrap<JoinRequestResponse>(requestJoin(member1.ctx, outing.id), 201)
        jr2 = await unwrap<JoinRequestResponse>(requestJoin(member2.ctx, outing.id), 201)
        jr3 = await unwrap<JoinRequestResponse>(requestJoin(pending.ctx, outing.id), 201)
        await unwrap(accept(host.ctx, jr1.id), 200)
        await unwrap(accept(host.ctx, jr2.id), 200)

    })
    test('host -> member -> 201, host -> pending -> 201, member -> member -> 201, member -> host -> 201', async () => {

        // host member
        const res1 = await unwrap<ConversationResponse>(startDM(host.ctx, {hiker_id:member1.id}), 201)
        expect(res1.dm_initiator).toBe(host.id)
        expect(res1.dm_status).toBe("pending")
        // host pending
        const res2 = await unwrap<ConversationResponse>(startDM(host.ctx, {hiker_id:pending.id}), 201)
        expect(res2.dm_initiator).toBe(host.id)
        expect(res2.dm_status).toBe("pending")
        // member member
        const res3 = await unwrap<ConversationResponse>(startDM(member1.ctx, {hiker_id:member2.id}), 201)
        expect(res3.dm_initiator).toBe(member1.id)
        expect(res3.dm_status).toBe("pending")
        // member host
        const res4 = await unwrap<ConversationResponse>(startDM(member2.ctx, {hiker_id:host.id}), 201)
        expect(res4.dm_initiator).toBe(member2.id)
        expect(res4.dm_status).toBe("pending")

    })
    test(`1-host->startDM->stranger->403, 
    2-member->startDM->pending->403,
    3-pending->startDM->member->403, 
    4-stranger->starDM->member->403,
    5-pending->startDM->host->403,
    6-self->startDM->self->400,
    7-member1->startDm->member2 / member2->startDm->member1-> same id
    `, async()=>{

        // 1
        await unwrap(startDM(host.ctx, {hiker_id:stranger.id}), 403)
        // 2
        await unwrap(startDM(member1.ctx, {hiker_id:pending.id}), 403)
        // 3
        await unwrap(startDM(pending.ctx, {hiker_id:member1.id}), 403)
        // 4
        await unwrap(startDM(stranger.ctx, {hiker_id:member1.id}), 403)
        // 5
        await unwrap(startDM(pending.ctx, {hiker_id:host.id}), 403)
        // 6
        await unwrap(startDM(member1.ctx, {hiker_id:member1.id}), 400)
        // 7
        const m1= await unwrap<ConversationResponse>(startDM(member1.ctx, {hiker_id:member2.id}), 201)
        const m2= await unwrap<ConversationResponse>(startDM(member2.ctx, {hiker_id:member1.id}), 201)
        expect(m1.id).toBe(m2.id)
    });
    test("member1 => startDM with member2 => member2 accepts => posts each other ", async ()=>{
        const dm = await unwrap<ConversationResponse>(startDM(member1.ctx, {hiker_id:member2.id}), 201)
        const premessage = await unwrap<Message>(postMessage(member1.ctx, dm.id, {body:"hello"}), 201)
        expect(premessage.body).toBe("hello")
        await unwrap<Message>(postMessage(member1.ctx, dm.id, {body:"hello"}), 403)
        await unwrapExpectNoContent(acceptDM(member1.ctx, dm.id), 403)
        await unwrap<Message>(postMessage(member2.ctx, dm.id, {body:"hello"}), 403)
        await unwrapExpectNoContent(acceptDM(member2.ctx, dm.id), 204)


        const message = await unwrap<Message>(postMessage(member1.ctx, dm.id, {body:"hello"}), 201)
        expect(message.body).toBe("hello")
        const reply  = await unwrap<Message>(postMessage(member2.ctx, dm.id, {body:"hi there"}), 201)
        expect(reply.body).toBe("hi there")
    });
    test("member1 => startDM with member2 => member2 declines", async ()=> {
        const dm = await unwrap<ConversationResponse>(startDM(member1.ctx, {hiker_id:member2.id}), 201)
        await unwrapExpectNoContent(declineDM(member2.ctx, dm.id), 204)
        await unwrapExpectNoContent(declineDM(member2.ctx, dm.id), 409)
        await unwrap<Message>(postMessage(member1.ctx, dm.id, {body:"hello"}), 403)
        await unwrapExpectNoContent(reopenDM(member1.ctx, dm.id), 403)
        await unwrapExpectNoContent(reopenDM(member2.ctx, dm.id), 204)
        const reply  = await unwrap<Message>(postMessage(member2.ctx, dm.id, {body:"hi there"}), 201)
        expect(reply.body).toBe("hi there")
        await unwrapExpectNoContent(reopenDM(member1.ctx, dm.id), 409)
        await unwrapExpectNoContent(declineDM(member1.ctx, dm.id), 204)
    });
    test("list chat", async () => {
        const dm = await unwrap<ConversationResponse>(startDM(member1.ctx, {hiker_id:member2.id}), 201)
        await unwrapExpectNoContent(acceptDM(member2.ctx, dm.id), 204)
        await unwrap(postMessage(member1.ctx, dm.id, {body:`hi ${member2.user?.name}`}), 201)
        await unwrap(postMessage(member2.ctx, dm.id, {body:`hi ${member1.user?.name}`}), 201)
        await unwrap(postMessage(member1.ctx, dm.id, {body:"can you pick me up on way to meet point?"}), 201)
        await unwrap(postMessage(member2.ctx, dm.id, {body:"sure."}), 201)
        const data = await unwrap<ListConversationsResponse>(listConversations(member1.ctx), 200)
        const dmp = data.conversations.find(d => d.id === dm.id)
        expect(dmp?.kind).toBe("dm")
        expect(dmp?.last_preview).toBe("sure.")
        expect(dmp?.title).toBe(member2.user?.name)
        const oc = data.conversations.find(c => c.id === outing.conversation_id)
        expect(oc?.kind).toBe("outing")
        expect(oc?.title).toBe(outing.title)
        const st = await unwrap<ListConversationsResponse>(listConversations(stranger.ctx), 200)
        expect(st.conversations.length).toBe(0)

    })



})