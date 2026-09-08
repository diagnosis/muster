import {expect, test} from '@playwright/test'
import {asUser,BASE} from "../fixtures";
import {accept, cancelOuting, createOuting, requestJoin, updateOuting} from "../api";
import {JoinRequestResponse, OutingResponse} from "../types";
import {unwrap, unwrapError} from "../envelope";
import {getNotificationsFor} from "../db";


test.describe("outing-crud actions", ()=> {
    test("create validation rejects bad inputs", async () =>{
        const { ctx: ctxHost} = await asUser(BASE)
        let res = await createOuting(ctxHost, {starts_at: new Date(Date.now()+60*60*1000).toISOString()})
        expect(res.status()).toBe(400)
        expect((await unwrapError(res)).message).toBe('outing has to be at least 24 hours in advance');

        res = await createOuting(ctxHost, {max_size:1})
        expect(res.status()).toBe(400)
        expect((await unwrapError(res)).message).toBe("outing size has to be at least 2")

        res = await createOuting(ctxHost, {difficulty:"extreme"})
        expect(res.status()).toBe(400)
        expect((await unwrapError(res)).message).toBe("invalid difficulty input")
    });
    test("patch merges fields, untouched fields survive", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)


        const updatedRes: OutingResponse =
            await unwrap<OutingResponse>(updateOuting(ctxHost, outing.id, {title:'Mt. Townsend', max_size:4}), 200)
        expect(updatedRes.title).toBe('Mt. Townsend')
        expect(updatedRes.max_size).toBe(4)
        expect(updatedRes.id).toBe(outing.id)
        expect(updatedRes.host_seats).toBe(outing.host_seats)
        expect(updatedRes.destination).toBe(outing.destination)

    });
    test("patch guards - nonhost patches", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const {ctx: ctxOutsider} = await asUser(BASE)

        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

        const res = await updateOuting(ctxOutsider, outing.id, {title:'recep in the house'})
        expect(res.status()).toBe(403)
    })
    test("patch guards - host patches after cancel", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

        let res = await cancelOuting(ctxHost, outing.id)
        expect(res.status()).toBe(200)

        res = await updateOuting(ctxHost, outing.id, {title:'Mt. Townsend', max_size:4})
        expect(res.status()).toBe(409)

    })

    test("patches guards - host patches starts at +1h", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)
        const res = await updateOuting(ctxHost, outing.id, {starts_at: new Date(Date.now()+60*60*1000).toISOString()})
        expect(res.status()).toBe(400)
    })

    test("already canceled outing", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

        let res = await cancelOuting(ctxHost, outing.id)
        expect(res.status()).toBe(200)

        res = await  cancelOuting(ctxHost, outing.id)
        expect(res.status()).toBe(409)
    })

    test("host cancels outing -> all members and pending hikers will be notified", async () => {
        const {ctx: ctxHost, id: hostID} = await asUser(BASE)
        const {ctx: ctxHiker1, id: hiker1ID} = await asUser(BASE)
        const {ctx: ctxHiker2, id: hiker2ID} = await asUser(BASE)
        const {ctx: ctxHiker3, id: hiker3ID} = await asUser(BASE)

        const o = await unwrap<OutingResponse>(await createOuting(ctxHost, {max_size:6, host_seats:3} ), 201)

        const jr1 = await unwrap<JoinRequestResponse>(await requestJoin(ctxHiker1, o.id, {role:'driver', seats_offered:3}), 201)
        const jr2 = await unwrap<JoinRequestResponse>(await requestJoin(ctxHiker2, o.id ,{role:'rider', guests:1}), 201)
        await unwrap<JoinRequestResponse>(await requestJoin(ctxHiker3, o.id ,{role:'rider', guests:1}), 201)

        await unwrap(await accept(ctxHost, jr1.id), 200)
        await unwrap(await accept(ctxHost, jr2.id), 200)

        await unwrap(await cancelOuting(ctxHost, o.id), 200)

        const notification1 = await getNotificationsFor(hiker1ID)
        expect(notification1.length).toBe(2)
        expect(notification1[1].kind).toBe('outing_cancelled')

        const notification2 = await getNotificationsFor(hiker2ID)
        expect(notification2.length).toBe(2)
        expect(notification2[1].kind).toBe('outing_cancelled')

        const notification3 = await getNotificationsFor(hiker3ID)
        expect(notification3.length).toBe(1)
        expect(notification3[0].kind).toBe('outing_cancelled')

        const hostNotification = await getNotificationsFor(hostID)
        expect(hostNotification.length).toBe(3)
        for(const n of hostNotification){
            expect(n.kind).not.toBe('outing_cancelled')
        }
    })
    test("host updates outing -> all members and pending hikers will be notified", async () => {
        const {ctx: ctxHost, id: hostID} = await asUser(BASE)
        const {ctx: ctxHiker1, id: hiker1ID} = await asUser(BASE)
        const {ctx: ctxHiker2, id: hiker2ID} = await asUser(BASE)
        const {ctx: ctxHiker3, id: hiker3ID} = await asUser(BASE)
        const {ctx: ctxHiker4, id: hiker4ID} = await asUser(BASE)
        const {ctx: ctxHiker5, id: hiker5ID} = await asUser(BASE)

        const o = await unwrap<OutingResponse>(await createOuting(ctxHost, {max_size:8, host_seats:3} ), 201)

        const jr1 = await unwrap<JoinRequestResponse>(await requestJoin(ctxHiker1, o.id, {role:'driver', seats_offered:4}), 201)
        const jr2 = await unwrap<JoinRequestResponse>(await requestJoin(ctxHiker2, o.id ,{role:'rider', guests:2}), 201)
        const jr3 = await unwrap<JoinRequestResponse>(await requestJoin(ctxHiker3, o.id ,{role:'rider', guests:1}), 201)
        const jr4 = await unwrap<JoinRequestResponse>(await requestJoin(ctxHiker4, o.id ,{role:'rider', guests:0}), 201)

        await unwrap(await accept(ctxHost, jr1.id), 200)
        await unwrap(await accept(ctxHost, jr2.id), 200)

        const updated = await unwrap<OutingResponse>(await updateOuting(ctxHost, o.id, {title:"updated title", host_seats:4, difficulty:"hard"}), 200)


        let hostNotification = await getNotificationsFor(hostID)

        expect(hostNotification.length).toBe(4)
        for(const n of hostNotification){
            expect(n.kind).not.toBe('outing_updated')
            expect(n.payload["outing_title"]).toBe(o.title)
        }

        const jr5 = await unwrap<JoinRequestResponse>(await requestJoin(ctxHiker5, updated.id), 201)

        await unwrap(await accept(ctxHost, jr3.id), 200)
        await unwrap(await accept(ctxHost, jr4.id), 200)

        hostNotification = await getNotificationsFor(hostID)
        expect(hostNotification.length).toBe(5)
        expect(hostNotification[4].payload["outing_title"]).toBe(updated.title)



        const notification1 = await getNotificationsFor(hiker1ID)
        expect(notification1.length).toBe(2)
        expect(notification1[1].kind).toBe('outing_updated')
        expect(notification1[1].payload["outing_title"]).toBe(updated.title)

        const notification2 = await getNotificationsFor(hiker2ID)
        expect(notification2.length).toBe(2)
        expect(notification2[1].kind).toBe('outing_updated')
        expect(notification2[1].payload["outing_title"]).toBe(updated.title)

        const notification3 = await getNotificationsFor(hiker3ID)
        expect(notification3.length).toBe(2)
        expect(notification3[0].kind).toBe('outing_updated')
        expect(notification3[0].payload["outing_title"]).toBe(updated.title)

        const notification4 = await getNotificationsFor(hiker4ID)
        expect(notification4.length).toBe(2)
        expect(notification4[0].kind).toBe('outing_updated')
        expect(notification4[0].payload["outing_title"]).toBe(updated.title)

    })
})