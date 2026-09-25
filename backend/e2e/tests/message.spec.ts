import {expect, test} from '@playwright/test'
import {asUser, BASE} from "../fixtures";
import {unwrap} from "../envelope";
import {accept, createOuting, requestJoin, postMessage, listMessages, deleteMessage} from "../api";
import {JoinRequestResponse, ListMessages, Message, OutingResponse} from "../types";

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

        await unwrap<Message>(postMessage(stranger.ctx, outing.conversation_id, {body:"yo"}), 403)

        const hostMessage = await unwrap<Message>(postMessage(host.ctx, outing.conversation_id, {body:`hi, ${member.user.name}`}), 201)
        expect(hostMessage.body).toBe(`hi, ${member.user.name}`)
        expect(hostMessage.seq).toBeGreaterThan(messageMember.seq)

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

    })
})