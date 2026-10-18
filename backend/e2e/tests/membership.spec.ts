import {APIRequestContext, expect, test} from '@playwright/test'
import { asUser,BASE } from "../fixtures";
import {
    accept,
    createOuting,
    decline, editJoinRequest,
    getDetail, getNotifications,
    listOutingRequests,
    removeMember,
    requestJoin, restoreRequest,
    withdraw
} from "../api";
import {
    DetailResponse,
    JoinRequestResponse,
    JoinRequestViewResponse, type NotificationsResponse,
    OutingResponse,
    RestoreRequestResponse
} from "../types";
import {unwrap, unwrapExpectNoContent} from "../envelope";
import {getNotificationsFor} from "../db";
import {unwatchFile} from "node:fs";


test.describe("membership", ()=> {
  test(`list upcoming request oldest first`, async () => {
      const {ctx: ctxHost, id: hostID} = await  asUser(BASE)
      const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker1} = await asUser(BASE)
      const joinReq1 = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker1, outing.id), 201)
      const hiker1Id = joinReq1.hiker_id
      const notifications = await getNotificationsFor(hostID)
      expect(notifications.length).toBe(1)
      expect(notifications[0].kind).toBe('join_request_created')

      const {ctx: ctxHiker2} = await asUser(BASE)
      const joinReq2 = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker2, outing.id), 201)
      const hiker2Id = joinReq2.hiker_id

      const {ctx: ctxHiker3} = await asUser(BASE)
      const joinReq3 = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker3, outing.id), 201)
      const hiker3Id = joinReq3.hiker_id

      const inbox = await unwrap<JoinRequestResponse[]>(listOutingRequests(ctxHost, outing.id, "requested"), 200)
      expect(inbox.length).toBe(3);
      expect(inbox[0].hiker_id).toBe(hiker1Id);   // oldest first
      expect(inbox[1].hiker_id).toBe(hiker2Id);
      expect(inbox[2].hiker_id).toBe(hiker3Id);
  });

  test('withdrawn re-request applies the NEW payload', async ()=> {
      const {ctx: ctxHost, id:hostID} = await  asUser(BASE)

      const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker} = await asUser(BASE)

      let res = await requestJoin(ctxHiker,outing.id, {role:'driver', 'seats_offered':3})
      expect(res.status()).toBe(201)
      res = await withdraw(ctxHiker, outing.id)
      expect(res.status()).toBe(200)

      const notifications = await getNotificationsFor(hostID)
      expect(notifications.length).toBe(2)
      expect(notifications[1].kind).toBe('join_request_withdrawn')


      const joinRequest =
          await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id, {role:'rider', 'seats_offered':0, guests:2}),201)
      expect(joinRequest.status).toBe('requested')
      expect(joinRequest.role).toBe('rider')
      expect(joinRequest.seats_offered).toBe(0)
      expect(joinRequest.guests).toBe(2)

  });

  test(`accept a member to outing`, async()=>{
      const {ctx: ctxHost} = await asUser(BASE)
      const outing = await  unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker, id: hikerID} = await asUser(BASE)
      const joinRequest = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id), 201)


      const acceptRes = await accept(ctxHost, joinRequest.id)
      expect(acceptRes.status()).toBe(200)

      const notifications = await getNotificationsFor(hikerID)
      expect(notifications.length).toBe(1)
      expect(notifications[0].kind).toBe('join_request_approved')


      const detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)

      expect(detail.roster.length).toBe(1)
      expect(detail.roster[0].hiker_id).toBe(joinRequest.hiker_id)
      expect(detail.my_request?.status).toBe('accepted')

  });
  test(`host declines request -> terminal user cannot request again`, async ()=> {
      const {ctx: ctxHost} = await asUser(BASE)
      const outing = await  unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker, id: hikerID} = await asUser(BASE)
      const joinRequest = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id), 201)

      const res = await decline(ctxHost, joinRequest.id)
      expect(res.status()).toBe(200)

      const notifications = await getNotificationsFor(hikerID)
      expect(notifications.length).toBe(1)
      expect(notifications[0].kind).toBe('join_request_declined')

      await expect.poll(async () => {
          const rows = await getNotificationsFor(hikerID)
          return rows[0]?.emailed_at
      }, { timeout: 10_000 }).not.toBeNull()

      const detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)
      expect(detail.my_request?.status).toBe('declined')

      const hikerRes = await requestJoin(ctxHiker, outing.id)
      expect(hikerRes.status()).toBe(409)

  });
  test(`host removes member from roster`, async () => {
      const {ctx: ctxHost} = await asUser(BASE)
      const outing = await  unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker, id: hikerID} = await asUser(BASE)
      const joinRequest = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id), 201)

      const acceptRes = await accept(ctxHost, joinRequest.id)
      expect(acceptRes.status()).toBe(200)

      let detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)
      expect(detail.roster.length).toBe(1)
      expect(detail.my_request?.status).toBe('accepted')
      let res = await removeMember(ctxHost, joinRequest.id)
      expect(res.status()).toBe(200)

      const notifications = await getNotificationsFor(hikerID)
      expect(notifications.length).toBe(2)
      expect(notifications[1].kind).toBe('member_removed')

      detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)
      expect(detail.roster.length).toBe(0)
      expect(detail.people_count).toBe(1)
      const rejoin = await requestJoin(ctxHiker, outing.id)
      expect(rejoin.status()).toBe(409)
      detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)
      expect(detail.my_request?.status).toBe('removed')


  });
  test('host cannot join own outing -> 400', async ()=> {
      const {ctx: ctxHost} = await asUser(BASE)
      const outing = await  unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const res = await requestJoin(ctxHost, outing.id)
      expect(res.status()).toBe(400)
  });
  test("re-request moves the hiker to the back of the queue", async ()=>{
      const {ctx: ctxHost} = await asUser(BASE)
      const m1 = await asUser(BASE)
      const m2 = await asUser(BASE)

      const o = await unwrap<OutingResponse>(createOuting(ctxHost), 201)
      const jr1 = await unwrap<JoinRequestResponse>(requestJoin(m1.ctx, o.id), 201)
      const jr2 = await unwrap<JoinRequestResponse>(requestJoin(m2.ctx, o.id), 201)
      await unwrap(withdraw(m1.ctx, o.id), 200)
      const jr3 = await unwrap<JoinRequestResponse>(requestJoin(m1.ctx, o.id), 201)

      expect(new Date(jr3.requested_at).getTime())
          .toBeGreaterThan(new Date(jr1.requested_at).getTime())
      const joinRequests = await unwrap<JoinRequestResponse[]>(listOutingRequests(ctxHost, o.id, ""), 200)
      expect(joinRequests.length).toBe(2)
      expect(joinRequests[0].id).toBe(jr2.id)
      expect(joinRequests[1].id).toBe(jr1.id)

      const detail = await unwrap<DetailResponse>(getDetail(m1.ctx, o.id), 200)
      expect(new Date(detail.my_request!.requested_at).getTime())
          .toBe(new Date(jr3.requested_at).getTime())

  })
    test("un-decline declined", async()=> {
        const host = await asUser(BASE)
        const hiker  = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const o = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const jr = await unwrap<JoinRequestResponse>(requestJoin(hiker.ctx, o.id), 201)
        const jr2 = await  unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, o.id), 201)
        let jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id, ""), 200)
        expect(jrs[0].id).toBe(jr.id)
        expect(jrs[1].id).toBe(jr2.id)
        expect(new Date(jrs[0].requested_at).getTime()).toBeLessThan(new Date(jrs[1].requested_at).getTime())
        const requestedAt = jrs[0].requested_at

        await unwrap(decline(host.ctx, jr.id), 200)
        await unwrap(accept(host.ctx, jr.id), 409)
        jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id, ""), 200)
        expect(jrs.length).toBe(1)

        const res = await unwrap<RestoreRequestResponse>(restoreRequest(host.ctx, jr.id), 200)
        expect(res.id).toBe(jr.id)
        jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id, ""), 200)

        expect(jrs[0].status).toBe("requested")
        expect(jrs[0].id).toBe(jr.id)
        expect(jrs[1].id).toBe(jr2.id)
        expect(new Date(jrs[0].requested_at).getTime()).toBeLessThan(new Date(jrs[1].requested_at).getTime())
        expect(jrs[0].requested_at).toBe(requestedAt)
    })
    test("un-remove removed", async () => {
        const host = await asUser(BASE)
        const hiker = await asUser(BASE)
        const hiker2 = await asUser(BASE)
        const o = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const jr = await unwrap<JoinRequestResponse>(requestJoin(hiker.ctx, o.id), 201)
        const jr2 = await unwrap<JoinRequestResponse>(requestJoin(hiker2.ctx, o.id), 201)

        let jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id, "requested"), 200)
        expect(jrs.map(j => j.id)).toEqual([jr.id, jr2.id])
        const requestedAt = jrs[0].requested_at

        await unwrap(accept(host.ctx, jr.id), 200)
        await unwrap(removeMember(host.ctx, jr.id), 200)

        const res = await unwrap<RestoreRequestResponse>(restoreRequest(host.ctx, jr.id), 200)
        expect(res.id).toBe(jr.id)

        // back in the queue, at the original place, with the original time
        jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id, "requested"), 200)
        expect(jrs.map(j => j.id)).toEqual([jr.id, jr2.id])
        expect(jrs[0].status).toBe("requested")
        expect(jrs[0].requested_at).toBe(requestedAt)

        // restore is not re-accept: nobody is on the roster
        const detail = await unwrap<DetailResponse>(getDetail(host.ctx, o.id), 200)
        expect(detail.roster).toHaveLength(0)
    })

    test("403 when non host attempts to restore", async () => {
        const host = await asUser(BASE)
        const hiker  = await asUser(BASE)
        const totalStranger = await asUser(BASE)
        const o = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const jr = await unwrap<JoinRequestResponse>(requestJoin(hiker.ctx, o.id), 201)
        await unwrap(removeMember(host.ctx, jr.id), 409)
        await unwrap(accept(host.ctx, jr.id),200)
        await unwrap(removeMember(host.ctx, jr.id), 200)

        await unwrap(restoreRequest(hiker.ctx, jr.id), 403)
        await unwrap(restoreRequest(totalStranger.ctx, jr.id), 403)

    })

    test("join request list filters by status", async () => {
        const host = await asUser(BASE)
        const members = await Promise.all([1, 2, 3, 4, 5, 6].map(() => asUser(BASE)))

        const o = await unwrap<OutingResponse>(createOuting(host.ctx, { max_size: 10, host_seats: 4 }), 201)
        const jrs: JoinRequestViewResponse[] = []
        for (const m of members) {
            jrs.push(await unwrap(requestJoin(m.ctx, o.id), 201))
        }


        let reqs = await unwrap<JoinRequestViewResponse[]>(listOutingRequests(host.ctx, o.id), 200)
        expect(ids(reqs)).toEqual(ids(jrs))

        await unwrap(decline(host.ctx, jrs[0].id), 200)
        await unwrap(decline(host.ctx, jrs[1].id), 200)
        reqs = await unwrap<JoinRequestViewResponse[]>(listOutingRequests(host.ctx, o.id, "declined"), 200)
        expect(ids(reqs)).toEqual([jrs[0].id, jrs[1].id])

        await unwrap(accept(host.ctx, jrs[2].id), 200)
        await unwrap(accept(host.ctx, jrs[3].id), 200)
        await unwrap(removeMember(host.ctx, jrs[2].id), 200)
        await unwrap(removeMember(host.ctx, jrs[3].id), 200)
        reqs = await unwrap<JoinRequestViewResponse[]>(listOutingRequests(host.ctx, o.id, "removed"), 200)
        expect(ids(reqs)).toEqual([jrs[2].id, jrs[3].id])

        // history never leaks into the default inbox
        reqs = await unwrap<JoinRequestViewResponse[]>(listOutingRequests(host.ctx, o.id), 200)
        expect(ids(reqs)).toEqual([jrs[4].id, jrs[5].id])

        await unwrap(listOutingRequests(host.ctx, o.id, "banana"), 400)
    });
  test("request order drivers first", async() => {
      const host = await asUser(BASE)
      const h1 = await asUser(BASE)
      const h2 = await asUser(BASE)
      const d1 = await asUser(BASE)
      const o = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
      const j1 = await  unwrap<JoinRequestResponse>(requestJoin(h1.ctx, o.id), 201)
      const j2 = await  unwrap<JoinRequestResponse>(requestJoin(h2.ctx, o.id), 201)
      const j3 = await  unwrap<JoinRequestResponse>(requestJoin(d1.ctx, o.id, {role:"driver", seats_offered:2}), 201)

      const reqs = await unwrap<JoinRequestViewResponse[]>(listOutingRequests(host.ctx, o.id), 200)
      expect(reqs.length).toBe(3)
      expect(ids(reqs)).toEqual([j3.id, j1.id, j2.id])

      await unwrap<JoinRequestResponse[]>(listOutingRequests(h2.ctx, o.id), 403)
      const details = await unwrap<DetailResponse>(getDetail(h2.ctx, o.id), 200)
      const names = details.waiting.map(w => w.name)
      expect(names).toEqual([d1.user.name, h1.user.name, h2.user.name])

  });
  test("edit ask more", async() => {
      const host = await asUser(BASE)
      const h1 = await asUser(BASE)
      const o = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
      const j1 = await  unwrap<JoinRequestResponse>(requestJoin(h1.ctx, o.id), 201)
      const beforeRequestedAt = j1.requested_at

      const updated = await unwrap<JoinRequestResponse>(editJoinRequest(h1.ctx, o.id, {guests:2}),200)
      let n = await unwrap<NotificationsResponse>(await getNotifications(host.ctx), 200)
      expect(n.unread_count).toBe(2)
      expect(n.notifications.filter(x => x.kind === 'join_request_updated')).toHaveLength(1)
      expect(Date.parse(updated.requested_at)).toBeGreaterThan(Date.parse(beforeRequestedAt))
  });
  test("remove guest will not update requested_at", async ()=>{
      const host = await asUser(BASE)
      const h1 = await asUser(BASE)
      const o = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
      const j1 = await  unwrap<JoinRequestResponse>(requestJoin(h1.ctx, o.id, {guests:2}), 201)
      const beforeRequestedAt = j1.requested_at

      const updated = await unwrap<JoinRequestResponse>(editJoinRequest(h1.ctx, o.id, {guests:1}),200)
      expect(Date.parse(updated.requested_at)).toBe(Date.parse(beforeRequestedAt))
  });
  test ("edit role to driver will move placement up", async ()=>{
      const host = await asUser(BASE)
      const h1 = await asUser(BASE)
      const h2 = await asUser(BASE)
      const futureDriver = await asUser(BASE)
      const o = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
      const j1 = await  unwrap<JoinRequestResponse>(requestJoin(h1.ctx, o.id), 201)
      const j2 = await  unwrap<JoinRequestResponse>(requestJoin(h2.ctx, o.id), 201)
      const j3 = await  unwrap<JoinRequestResponse>(requestJoin(futureDriver.ctx, o.id), 201)
      let reqs = await unwrap<JoinRequestViewResponse[]>(listOutingRequests(host.ctx, o.id), 200)
      expect(reqs.length).toBe(3)
      expect(ids(reqs)).toEqual([j1.id, j2.id, j3.id])

      await unwrap<JoinRequestResponse>(editJoinRequest(futureDriver.ctx, o.id, {role:'driver', seats_offered:2}), 200)
      reqs = await unwrap<JoinRequestViewResponse[]>(listOutingRequests(host.ctx, o.id), 200)
      expect(reqs.length).toBe(3)
      expect(ids(reqs)).toEqual([j3.id, j1.id, j2.id])
  });
  test("accepted hiker attempt to edit -> 409", async()=>{
      const host = await asUser(BASE)
      const h1 = await asUser(BASE)
      const o = await unwrap<OutingResponse>(createOuting(host.ctx, {max_size:10, host_seats:4}), 201)
      const j1 = await  unwrap<JoinRequestResponse>(requestJoin(h1.ctx, o.id, {guests:2}), 201)
      await unwrap(accept(host.ctx, j1.id), 200)
      await unwrap<JoinRequestResponse>(editJoinRequest(h1.ctx, o.id, {note:'i can be driver'}), 409)
  });
  test("hiker with no request->404", async () => {
      const host = await asUser(BASE)
      const h1 = await asUser(BASE)
      const alien = await asUser(BASE)
      const o = await unwrap<OutingResponse>(createOuting(host.ctx, {max_size:10, host_seats:4}), 201)
      await  unwrap<JoinRequestResponse>(requestJoin(h1.ctx, o.id, {guests:2}), 201)
      await unwrap<JoinRequestResponse>(editJoinRequest(alien.ctx, o.id, {note:'i can be driver'}), 404)
  })


})

const ids = (list: JoinRequestViewResponse[]) => list.map(r => r.id)