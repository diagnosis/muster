import {expect, test} from '@playwright/test'
import {asUser, BASE} from "../fixtures";
import {unwrap, unwrapExpectNoContent} from "../envelope";
import {
    accept,
    createOuting,
    requestJoin,
    postMessage,
    listMessages,
    deleteMessage,
    markRead,
    listConversations
} from "../api";
import {JoinRequestResponse, ListConversationsResponse, ListMessages, Message, OutingResponse} from "../types";

test.describe("message api", ()=> {
    test(`member posts -> 201; stranger posts->403; 
    host posts-> 201 -> member list -> 200 -> stranger lists -> 403`, async () => {
        const host = await asUser(BASE)
        const member =  await asUser(BASE)
        const stranger = await asUser(BASE)

        const outing = await unwrap<OutingResponse>(createOuting(host.ctx, {max_size:8, host_seats:5}), 201)
        const jr = await unwrap<JoinRequestResponse>(requestJoin(member.ctx, outing.id), 201)
        await accept(host.ctx, jr.id)

        const messageMember = await unwrap<Message>(postMessage(member.ctx, outing.conversation_id, {body:"hello"} ), 201)
        expect(messageMember.body).toBe("hello")
        expect(messageMember.seq).toBeTruthy()
        await unwrapExpectNoContent(markRead(host.ctx, outing.conversation_id, {seq:messageMember.seq}), 204)
        await unwrap<Message>(postMessage(stranger.ctx, outing.conversation_id, {body:"yo"}), 403)

        const hostMessage = await unwrap<Message>(postMessage(host.ctx, outing.conversation_id, {body:`hi, ${member.user.name}`}), 201)
        expect(hostMessage.body).toBe(`hi, ${member.user.name}`)
        expect(hostMessage.seq).toBeGreaterThan(messageMember.seq)
        await unwrap(markRead(stranger.ctx, outing.conversation_id, {seq:hostMessage.seq}), 403)

        let data = await unwrap<ListMessages>(listMessages(member.ctx, outing.conversation_id), 200)
        expect(data.messages.length).toBe(2)
        expect(data.messages[0].seq).toBeLessThan(data.messages[1].seq)
        await unwrap<ListMessages>(listMessages(stranger.ctx, outing.conversation_id), 403)

        await unwrap(deleteMessage(member.ctx, hostMessage.id), 403)
        const noContentSuccess  = await deleteMessage(host.ctx, messageMember.id)
        expect(noContentSuccess.status()).toBe(204)

        data= await unwrap<ListMessages>(listMessages(member.ctx, outing.conversation_id), 200)
        expect(data.messages.length).toBe(1)


        await unwrap<Message>(postMessage(member.ctx, outing.conversation_id, {body:""} ), 400)
        await unwrap<Message>(postMessage(member.ctx, outing.conversation_id, {body:"a".repeat(501)} ), 400)
        await unwrap<Message>(postMessage(member.ctx, outing.conversation_id, {body:"a".repeat(500)} ), 201)
        await unwrap<Message>(postMessage(member.ctx, "random-conversation", {body:"hello"} ),400)
        await unwrap<Message>(postMessage(member.ctx, crypto.randomUUID(), {body:"hello"} ),404)

    });
    test("unread counts", async() => {
        const host = await asUser(BASE)
        const member =  await asUser(BASE)

        const outing = await unwrap<OutingResponse>(createOuting(host.ctx, {max_size:8, host_seats:5}), 201)
        const jr = await unwrap<JoinRequestResponse>(requestJoin(member.ctx, outing.id), 201)
        await accept(host.ctx, jr.id)
        await postMessage(member.ctx, outing.conversation_id, {body:"hello host"})
        await postMessage(member.ctx, outing.conversation_id, {body:"are you there"})
        const lastMessage = await unwrap<Message>(postMessage(member.ctx, outing.conversation_id, {body:"how is it going?"}), 201)
        let data = await unwrap<ListConversationsResponse>(listConversations(host.ctx), 200)
        let conv = data.conversations.find(c => c.id === outing.conversation_id)
        expect(conv).toBeDefined()
        expect(conv?.unread_count).toBe(3)
        await postMessage(host.ctx, outing.conversation_id, {body:"hi there"})
        data = await unwrap<ListConversationsResponse>(listConversations(host.ctx), 200)
        conv = data.conversations.find(c => c.id === outing.conversation_id)
        expect(conv).toBeDefined()
        expect(conv?.unread_count).toBe(3)

        await unwrapExpectNoContent(markRead(host.ctx, outing.conversation_id, {seq:lastMessage.seq}), 204)
        data = await unwrap<ListConversationsResponse>(listConversations(host.ctx), 200)
        conv = data.conversations.find(c => c.id === outing.conversation_id)
        expect(conv).toBeDefined()
        expect(conv?.unread_count).toBe(0)

        data = await unwrap<ListConversationsResponse>(listConversations(member.ctx), 200)
        conv = data.conversations.find(c => c.id === outing.conversation_id)
        expect(conv).toBeDefined()
        expect(conv?.unread_count).toBe(1)
    });
    test("unread count on accept will mark last message as read", async () => {
        const host = await asUser(BASE)
        const member1 = await asUser(BASE)
        const member2 = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(host.ctx, {max_size:8, host_seats:5}), 201)
        const jr = await unwrap<JoinRequestResponse>(requestJoin(member1.ctx, outing.id), 201)
        const jr2 = await unwrap<JoinRequestResponse>(requestJoin(member2.ctx, outing.id), 201)
        await accept(host.ctx, jr.id)
        await postMessage(member1.ctx, outing.conversation_id, {body:"hello host"})
        await postMessage(member1.ctx, outing.conversation_id, {body:"are you there"})
        await postMessage(host.ctx, outing.conversation_id, {body:"hi, there"})
        await accept(host.ctx, jr2.id)
        let data = await unwrap<ListConversationsResponse>(listConversations(member2.ctx), 200)
        let conv = data.conversations.find(c=>c.id===outing.conversation_id)
        expect(conv).toBeDefined()
        expect(conv?.unread_count).toBe(0)
        await postMessage(host.ctx, outing.conversation_id, {body:"welcome to the pack mister"})
        data = await unwrap<ListConversationsResponse>(listConversations(member2.ctx), 200)
        conv = data.conversations.find(c=>c.id===outing.conversation_id)
        expect(conv).toBeDefined()
        expect(conv?.unread_count).toBe(1)


    })

})