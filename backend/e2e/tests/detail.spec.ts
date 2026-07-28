import {expect, request, test} from '@playwright/test'
import {asUser, BASE} from "../fixtures";
import {accept, createOuting, getDetail, requestJoin, updateOuting} from "../api";
import {DetailResponse, JoinRequestResponse, OutingResponse} from "../types";
import {unwrap} from "../envelope";

test.describe("detail", ()=>{
    test("anonymous full assembly", async () =>{
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)
        const {ctx: ctxRider} = await asUser(BASE)
        const joinRequest = await unwrap<JoinRequestResponse>(requestJoin(ctxRider, outing.id), 201)

        const res = await accept(ctxHost, joinRequest.id)
        expect(res.status()).toBe(200)

        const ctxAnonymous = await request.newContext({baseURL:BASE})
        const detail = await unwrap<DetailResponse>(getDetail(ctxAnonymous, outing.id), 200)


        expect(detail.roster.length).toBe(1)
        expect(detail.host.name).toBe('Test User')
        expect(detail.seat_capacity).toBe(2)
        expect(detail.people_count).toBe(2)
        expect(detail.seats_short).toBe(0)
        expect(detail.spots_left).toBe(0)
        expect(detail.my_request).toBeUndefined()
    });
    test("shrink surfaces shortage", async ()=> {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)
        const res = await updateOuting(ctxHost, outing.id, {host_seats:0})
        expect(res.status()).toBe(200)

        const detail = await unwrap<DetailResponse>(getDetail(ctxHost, outing.id), 200)

        expect(detail.seats_short).toBe(1)
        expect(detail.spots_left).toBe(0)
    });
    test("empty roster", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)
        const detail = await unwrap<DetailResponse>(getDetail(ctxHost, outing.id), 200)
        expect(detail.roster.length).toBe(0)
        expect(Array.isArray(detail.roster)).toBe(true)
    })
})