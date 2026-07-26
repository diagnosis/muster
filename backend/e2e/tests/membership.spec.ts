import {expect, test} from '@playwright/test'
import { asUser } from "../fixtures";
import {accept, createOuting, decline, getDetail, removeMember, requestJoin, withdraw} from "../api";
import {DetailResponse, JoinRequestResponse, OutingResponse} from "../types";

const BASE = 'http://localhost:8088'

test.describe("membership", ()=> {
  test(`list upcoming request oldest first`, async () => {
      const {ctx: ctxHost} = await  asUser(BASE)

      let res =await createOuting(ctxHost)
      expect(res.status()).toBe(201)

      const outingData: OutingResponse = (await res.json())['data']

      const {ctx: ctxHiker1} = await asUser(BASE)
      const {ctx: ctxHiker2} = await asUser(BASE)
      const {ctx: ctxHiker3} = await asUser(BASE)

      const firstReqRes = await requestJoin(ctxHiker1, outingData.id)
      expect(firstReqRes.status()).toBe(201)
      const hiker1: JoinRequestResponse = (await firstReqRes.json())['data']
      const hiker1Id = hiker1.hiker_id

      const secondReqRes = await requestJoin(ctxHiker2, outingData.id)
      expect(secondReqRes.status()).toBe(201)
      const hiker2: JoinRequestResponse = (await secondReqRes.json())['data']
      const hiker2Id = hiker2.hiker_id

      const thirdReqRes = await requestJoin(ctxHiker3, outingData.id)
      expect(thirdReqRes.status()).toBe(201)
      const hiker3: JoinRequestResponse = (await thirdReqRes.json())['data']
      const hiker3Id = hiker3.hiker_id


      res = await ctxHost.get(`/api/outings/${outingData.id}/requests`)
      expect(res.status()).toBe(200)
      const inbox = (await res.json()).data;
      expect(inbox.length).toBe(3);
      expect(inbox[0].hiker_id).toBe(hiker1Id);   // oldest first
      expect(inbox[1].hiker_id).toBe(hiker2Id);
      expect(inbox[2].hiker_id).toBe(hiker3Id);
  });

  test('withdrawn re-request keeps ORIGINAL role/seats (v0 spec — scheduled to change, see parking lot: v1-early)', async ()=> {
      const {ctx: ctxHost} = await  asUser(BASE)

      let res =await createOuting(ctxHost)
      expect(res.status()).toBe(201)

      const outingData: OutingResponse = (await res.json())['data']
      expect(outingData.id).toBeTruthy()

      const {ctx: ctxHiker} = await asUser(BASE)
      let hikerRes = await requestJoin(ctxHiker,outingData.id, {role:'driver', 'seats_offered':3})
      expect(hikerRes.status()).toBe(201)
      hikerRes = await withdraw(ctxHiker, outingData.id)
      expect(hikerRes.status()).toBe(200)
      hikerRes = await requestJoin(ctxHiker, outingData.id)
      expect(hikerRes.status()).toBe(201)
      const hiker : JoinRequestResponse =  (await hikerRes.json())['data']
      expect(hiker.status).toBe('requested')
      expect(hiker.role).toBe('driver')
      expect(hiker.seats_offered).toBe(3)

  });

  test(`accept a member to outing`, async()=>{
      const {ctx: ctxHost} = await asUser(BASE)

      let res = await createOuting(ctxHost)
      expect(res.status()).toBe(201)

      const outingData : OutingResponse = (await res.json())['data']
      expect(outingData.id).toBeTruthy()

      const {ctx: ctxHiker} = await asUser(BASE)
      let hikerRes = await requestJoin(ctxHiker, outingData.id)
      expect(hikerRes.status()).toBe(201)
      const hikerResData: JoinRequestResponse = (await hikerRes.json())['data']

      const acceptRes = await accept(ctxHost, hikerResData.id)
      expect(acceptRes.status()).toBe(200)

      res = await getDetail(ctxHiker, outingData.id)
      expect(res.status()).toBe(200)
      const detail:DetailResponse = (await res.json())['data']
      expect(detail.roster.length).toBe(1)
      expect(detail.roster[0].hiker_id).toBe(hikerResData.hiker_id)
      expect(detail.my_request?.status).toBe('accepted')

  });
  test(`host declines request -> terminal user cannot request again`, async ()=> {
      const {ctx: ctxHost} = await asUser(BASE)

      let res = await createOuting(ctxHost)
      expect(res.status()).toBe(201)

      const outingData : OutingResponse = (await res.json())['data']
      expect(outingData.id).toBeTruthy()

      const {ctx: ctxHiker} = await asUser(BASE)
      let hikerRes = await requestJoin(ctxHiker, outingData.id)
      expect(hikerRes.status()).toBe(201)
      const hikerResData: JoinRequestResponse = (await hikerRes.json())['data']

      res = await decline(ctxHost, hikerResData.id)
      expect(res.status()).toBe(200)

      res = await getDetail(ctxHiker, outingData.id)
      expect(res.status()).toBe(200)
      const detail:DetailResponse = (await res.json())['data']
      expect(detail.my_request?.status).toBe('declined')

      hikerRes = await requestJoin(ctxHiker, outingData.id)
      expect(hikerRes.status()).toBe(409)

  });
  test(`host removes member from roster`, async () => {
      const {ctx: ctxHost} = await asUser(BASE)

      let res = await createOuting(ctxHost)
      expect(res.status()).toBe(201)

      const outingData: OutingResponse = (await res.json())['data']
      expect(outingData.id).toBeTruthy()

      const {ctx: ctxHiker} = await asUser(BASE)
      let hikerRes = await requestJoin(ctxHiker, outingData.id)
      expect(hikerRes.status()).toBe(201)
      const hikerResData: JoinRequestResponse = (await hikerRes.json())['data']


      const acceptRes = await accept(ctxHost, hikerResData.id)
      expect(acceptRes.status()).toBe(200)

      res = await getDetail(ctxHiker, outingData.id)
      expect(res.status()).toBe(200)
      let detail:DetailResponse = (await res.json())['data']
      expect(detail.roster.length).toBe(1)

      res = await removeMember(ctxHost, hikerResData.id)
      expect(res.status()).toBe(200)
      res = await getDetail(ctxHiker, outingData.id)
      expect(res.status()).toBe(200)
      detail = (await res.json())['data']
      expect(detail.roster.length).toBe(0)
      expect(detail.people_count).toBe(1)

  });
  test('host cannot join own outing -> 400', async ()=> {
      const {ctx: ctxHost} = await asUser(BASE)
      let res = await createOuting(ctxHost)
      const outingData: OutingResponse = (await res.json())['data']

      res = await requestJoin(ctxHost, outingData.id)
      expect(res.status()).toBe(400)
  })

})

