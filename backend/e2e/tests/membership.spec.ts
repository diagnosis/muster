import {expect, test} from '@playwright/test'
import { asUser,BASE } from "../fixtures";
import {
    accept,
    createOuting,
    decline,
    getDetail,
    listOutingRequests,
    removeMember,
    requestJoin, restoreRequest,
    withdraw
} from "../api";
import {DetailResponse, JoinRequestResponse, OutingResponse, RestoreRequestResponse} from "../types";
import {unwrap, unwrapExpectNoContent} from "../envelope";
import {getNotificationsFor} from "../db";


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

      const inbox = await unwrap<JoinRequestResponse[]>(listOutingRequests(ctxHost, outing.id), 200)
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
      const joinRequests = await unwrap<JoinRequestResponse[]>(listOutingRequests(ctxHost, o.id), 200)
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
        let jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id), 200)
        expect(jrs[0].id).toBe(jr.id)
        expect(jrs[1].id).toBe(jr2.id)
        expect(new Date(jrs[0].requested_at).getTime()).toBeLessThan(new Date(jrs[1].requested_at).getTime())
        const requestedAt = jrs[0].requested_at

        await unwrap(decline(host.ctx, jr.id), 200)
        await unwrap(accept(host.ctx, jr.id), 409)
        jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id), 200)
        expect(jrs.length).toBe(1)

        const res = await unwrap<RestoreRequestResponse>(restoreRequest(host.ctx, jr.id), 200)
        expect(res.id).toBe(jr.id)
        jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id), 200)

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

        let jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id), 200)
        expect(jrs.map(j => j.id)).toEqual([jr.id, jr2.id])
        const requestedAt = jrs[0].requested_at

        await unwrap(accept(host.ctx, jr.id), 200)
        await unwrap(removeMember(host.ctx, jr.id), 200)

        const res = await unwrap<RestoreRequestResponse>(restoreRequest(host.ctx, jr.id), 200)
        expect(res.id).toBe(jr.id)

        // back in the queue, at the original place, with the original time
        jrs = await unwrap<JoinRequestResponse[]>(listOutingRequests(host.ctx, o.id), 200)
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


})

