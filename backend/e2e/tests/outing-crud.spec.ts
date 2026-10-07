import {expect, test} from '@playwright/test'
import {asUser,BASE} from "../fixtures";
import {accept, cancelOuting, createOuting, getDetail, requestJoin, updateOuting} from "../api";
import {DetailResponse, JoinRequestResponse, OutingResponse} from "../types";
import {unwrap, unwrapError} from "../envelope";
import {getNotificationsFor} from "../db";
import {hostname} from "node:os";
import {at, plusHours, sameTime} from "../utils/date";


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

    });
    test("host creates outing with end before start date", async () => {
        const host = await asUser(BASE)
        await unwrap<OutingResponse>(createOuting(host.ctx, {starts_at:at(3,6), ends_at:at(2, 18)}), 400)
    })
    test("host creates outing with ends and starts date same", async ()=>{
        const host = await asUser(BASE)
        await unwrap<OutingResponse>(createOuting(host.ctx, {starts_at:at(3,6), ends_at:at(3, 6)}), 400)
    })
    test("host creates outing with longer than 14 days", async()=> {
        const host = await asUser(BASE)
        await unwrap(createOuting(host.ctx,{starts_at:at(3,6), ends_at:at(17,8)}), 400)
    })
    test("host creates outing and updates starts and ends date to validate backend codes", async() => {
        const host = await asUser(BASE)
        let o =
            await unwrap<OutingResponse>(createOuting(host.ctx, {starts_at:at(3,6), ends_at:at(3, 18)}), 201)

        // starts moves pass existing end
        await unwrap(updateOuting(host.ctx, o.id,{starts_at:at(4,8)}), 400)
        // ends moves before existing starts date
        await unwrap(updateOuting(host.ctx, o.id, {ends_at:at(2, 18)}), 400)
        // end pushed passed the 14 days limit
        await unwrap(updateOuting(host.ctx, o.id, {ends_at:at(17, 8)}), 400)
        // valid end
        await unwrap(updateOuting(host.ctx, o.id, {ends_at:at(10, 9)}), 200)
        // end date no mention
        await unwrap(updateOuting(host.ctx, o.id, {title:"updated"}), 200)
        // valid start and end update
        o = await unwrap<OutingResponse>(createOuting(host.ctx, {starts_at:at(3,6), ends_at:at(3, 18)}), 201)
        const newEnd = at(10, 9)
        let updated = await unwrap<OutingResponse>(updateOuting(host.ctx, o.id, {ends_at: newEnd}), 200)
        expect(new Date(updated.ends_at!).getTime()).toBe(new Date(newEnd).getTime())

        updated = await unwrap<OutingResponse>(updateOuting(host.ctx, o.id, {title: "updated"}), 200)
        expect(updated.title).toBe("updated")
        expect(sameTime(updated.ends_at!, newEnd)).toBe(true)
    })
    test("host updates both start and end date", async () => {
        const host = await asUser(BASE)
        let o = await unwrap<OutingResponse>
        (createOuting(host.ctx, {starts_at:at(3,6), ends_at:at(3, 18)}), 201)
        // small shift
        let newEnd = at(3, 20)
        let newStart = at(3, 10)
        let updated = await unwrap<OutingResponse>(updateOuting(host.ctx, o.id, {starts_at:newStart, ends_at:newEnd}), 200)
        expect(new Date(updated.ends_at!).getTime()).toBe(new Date(newEnd).getTime())
        expect(new Date(updated.starts_at!).getTime()).toBe(new Date(newStart).getTime())
        // big shift
        o = await unwrap<OutingResponse>
        (createOuting(host.ctx, {starts_at:at(3,6), ends_at:at(3, 18)}), 201)
        newEnd = at(25, 20)
        newStart = at(25, 8)
        updated = await unwrap<OutingResponse>(updateOuting(host.ctx, o.id, {starts_at:newStart, ends_at:newEnd}), 200)
        expect(new Date(updated.ends_at!).getTime()).toBe(new Date(newEnd).getTime())
        expect(new Date(updated.starts_at!).getTime()).toBe(new Date(newStart).getTime())
        //start after end
        const originalStart = at(3, 6)
        const originalEnd = at(3, 28)
        o = await unwrap<OutingResponse>
        (createOuting(host.ctx, {starts_at:originalStart, ends_at:originalEnd}), 201)
        newEnd = at(23, 20)
        newStart= at(24, 6)
        await unwrap(updateOuting(host.ctx, o.id, {starts_at:newStart, ends_at:newEnd}), 400)
        const after = await unwrap<DetailResponse>(getDetail(host.ctx, o.id), 200)
        expect(new Date(after.outing.starts_at).getTime()).toBe(new Date(originalStart).getTime())
        expect(new Date(after.outing.ends_at!).getTime()).toBe(new Date(originalEnd).getTime())

        // exactly 14 days: accepted
        const s = at(5, 6)
        updated = await unwrap<OutingResponse>(
            updateOuting(host.ctx, o.id, {starts_at: s, ends_at: plusHours(s, 14 * 24)}), 200)

// one hour over: rejected
        await unwrap(updateOuting(host.ctx, o.id, {starts_at: s, ends_at: plusHours(s, 14 * 24 + 1)}), 400)

    })

    test("host creates outing with no end time", async () => {
        const host = await asUser(BASE)
        const o = await unwrap<OutingResponse>(createOuting(host.ctx, {starts_at: at(3, 6)}), 201)
        expect(o.ends_at).toBeNull()
    })

    test("host creates outing with an end time", async () => {
        const host = await asUser(BASE)
        const end = at(3, 18)
        const o = await unwrap<OutingResponse>(createOuting(host.ctx, {starts_at: at(3, 6), ends_at: end}), 201)
        expect(new Date(o.ends_at!).getTime()).toBe(new Date(end).getTime())
    })

    test("host adds an end time to an outing that had none", async () => {
        const host = await asUser(BASE)
        const o = await unwrap<OutingResponse>(createOuting(host.ctx, {starts_at: at(3, 6)}), 201)
        const detail  = await unwrap<DetailResponse>(getDetail(host.ctx, o.id), 200)
        expect(detail.outing.ends_at).toBeNull()
        const end = at(3, 18)
        const updated = await unwrap<OutingResponse>(updateOuting(host.ctx, o.id, {ends_at: end, clear_ends_at:false}), 200)
        expect(new Date(updated.ends_at!).getTime()).toBe(new Date(end).getTime())
    })
    test("outing with an end, check wins", async()=> {
        const host = await asUser(BASE)
        const o =
            await unwrap<OutingResponse>(createOuting(host.ctx, {starts_at: at(3, 6), ends_at:at(3, 22)}), 201)
        await unwrap(updateOuting(host.ctx, o.id, {clear_ends_at: true, ends_at: at(3, 20)}), 200)
        const detail = await unwrap<DetailResponse>(getDetail(host.ctx, o.id), 200)
        expect(detail.outing.ends_at).toBeNull()

    })
    test("create with clear_ends_at returns 400", async ()=> {
        const host = await asUser(BASE)
        await unwrap(createOuting(host.ctx, {clear_ends_at:true}), 400)
    })

})

