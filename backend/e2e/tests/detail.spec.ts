import {expect, request, test} from '@playwright/test'
import {asUser} from "../fixtures";
import {accept, createOuting, getDetail, requestJoin, updateOuting} from "../api";
import {DetailResponse, JoinRequestResponse, OutingResponse} from "../types";

const BASE = "http://localhost:8088"
test.describe("detail", ()=>{
    test("anonymous full assembly", async () =>{
        const {ctx: ctxHost} = await asUser(BASE)
        let res = await createOuting(ctxHost)
        expect(res.status()).toBe(201)

        const outingData: OutingResponse = (await res.json())['data']
        const {ctx: ctxRider} = await asUser(BASE)
        res = await requestJoin(ctxRider, outingData.id)
        expect(res.status()).toBe(201)
        const joinRequestData: JoinRequestResponse = (await res.json())['data']

        res = await accept(ctxHost, joinRequestData.id)
        expect(res.status()).toBe(200)

        const ctxAnonymous = await request.newContext({baseURL:BASE})
        res = await getDetail(ctxAnonymous, outingData.id)
        expect(res.status()).toBe(200)
        const details: DetailResponse = (await res.json())['data']

        expect(details.roster.length).toBe(1)
        expect(details.host.name).toBe('Test User')
        expect(details.seat_capacity).toBe(2)
        expect(details.people_count).toBe(2)
        expect(details.seats_short).toBe(0)
        expect(details.spots_left).toBe(0)
        expect(details.my_request).toBeUndefined()
    });
    test("shrink surfaces shortage", async ()=> {
        const {ctx: ctxHost} = await asUser(BASE)
        let res = await createOuting(ctxHost)
        expect(res.status()).toBe(201)

        const outingData: OutingResponse = (await res.json())['data']
        res = await updateOuting(ctxHost, outingData.id, {host_seats:0})
        expect(res.status()).toBe(200)

        res = await getDetail(ctxHost, outingData.id)
        expect(res.status()).toBe(200)
        const details: DetailResponse = (await res.json())['data']

        expect(details.seats_short).toBe(1)
        expect(details.spots_left).toBe(0)
    });
    test("empty roster", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        let res = await createOuting(ctxHost)
        expect(res.status()).toBe(201)

        const outingData: OutingResponse = (await res.json())['data']

        res = await getDetail((await request.newContext({baseURL:BASE})), outingData.id)
        expect(res.status()).toBe(200)

        const detail : DetailResponse = (await res.json())['data']
        expect(detail.roster.length).toBe(0)
        expect(Array.isArray(detail.roster)).toBe(true)
    })
})