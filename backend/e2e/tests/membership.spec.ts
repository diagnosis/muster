import {expect, test} from '@playwright/test'
import { asUser,BASE } from "../fixtures";
import {
    accept,
    createOuting,
    decline,
    getDetail,
    listOutingRequests,
    removeMember,
    requestJoin,
    withdraw
} from "../api";
import {DetailResponse, JoinRequestResponse, OutingResponse} from "../types";
import {unwrap} from "../envelope";


test.describe("membership", ()=> {
  test(`list upcoming request oldest first`, async () => {
      const {ctx: ctxHost} = await  asUser(BASE)
      const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker1} = await asUser(BASE)
      const joinReq1 = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker1, outing.id), 201)
      const hiker1Id = joinReq1.hiker_id

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

  test('withdrawn re-request keeps ORIGINAL role/seats (v0 spec — scheduled to change, see parking lot: v1-early)', async ()=> {
      const {ctx: ctxHost} = await  asUser(BASE)

      const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker} = await asUser(BASE)

      let res = await requestJoin(ctxHiker,outing.id, {role:'driver', 'seats_offered':3})
      expect(res.status()).toBe(201)
      res = await withdraw(ctxHiker, outing.id)
      expect(res.status()).toBe(200)

      const joinRequest =  await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id),201)
      expect(joinRequest.status).toBe('requested')
      expect(joinRequest.role).toBe('driver')
      expect(joinRequest.seats_offered).toBe(3)

  });

  test(`accept a member to outing`, async()=>{
      const {ctx: ctxHost} = await asUser(BASE)
      const outing = await  unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker} = await asUser(BASE)
      const joinRequest = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id), 201)


      const acceptRes = await accept(ctxHost, joinRequest.id)
      expect(acceptRes.status()).toBe(200)

      const detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)

      expect(detail.roster.length).toBe(1)
      expect(detail.roster[0].hiker_id).toBe(joinRequest.hiker_id)
      expect(detail.my_request?.status).toBe('accepted')

  });
  test(`host declines request -> terminal user cannot request again`, async ()=> {
      const {ctx: ctxHost} = await asUser(BASE)
      const outing = await  unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker} = await asUser(BASE)
      const joinRequest = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id), 201)

      const res = await decline(ctxHost, joinRequest.id)
      expect(res.status()).toBe(200)

      const detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)
      expect(detail.my_request?.status).toBe('declined')

      const hikerRes = await requestJoin(ctxHiker, outing.id)
      expect(hikerRes.status()).toBe(409)

  });
  test(`host removes member from roster`, async () => {
      const {ctx: ctxHost} = await asUser(BASE)
      const outing = await  unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const {ctx: ctxHiker} = await asUser(BASE)
      const joinRequest = await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id), 201)


      const acceptRes = await accept(ctxHost, joinRequest.id)
      expect(acceptRes.status()).toBe(200)

      let detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)
      expect(detail.roster.length).toBe(1)

      let res = await removeMember(ctxHost, joinRequest.id)
      expect(res.status()).toBe(200)
      detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id), 200)
      expect(detail.roster.length).toBe(0)
      expect(detail.people_count).toBe(1)

  });
  test('host cannot join own outing -> 400', async ()=> {
      const {ctx: ctxHost} = await asUser(BASE)
      const outing = await  unwrap<OutingResponse>(createOuting(ctxHost), 201)

      const res = await requestJoin(ctxHost, outing.id)
      expect(res.status()).toBe(400)
  })

})

