import {expect, request, test} from "@playwright/test";
import {asUser, BASE} from "../fixtures";
import {unwrap} from "../envelope";
import {accept, cancelOuting, createOuting, getHiker, requestJoin} from "../api";
import {HikerCardResponse, JoinRequestResponse, OutingResponse, PublicProfileResponse} from "../types";
import {randomUUID} from "node:crypto";
import {setOutingTimes} from "../db";


test.describe('hiker public profile card', ()=>{
    test('register and check (new hiker)', async ()=>{
        const hiker  = await asUser(BASE, {experience:"intermediate", bio:"cold brew", home_area:"serik"})
        const viewer = await asUser(BASE)
        const hcr = await unwrap<HikerCardResponse>(getHiker(viewer.ctx, hiker.id), 200)
        expect(hcr).toEqual({
            id: hiker.id,
            name: hiker.user.name,
            experience: 'intermediate',
            home_area: 'serik',
            bio: 'cold brew',
            hosted_count: 0,
            went_count: 0,
            member_since: expect.any(String),
            can_dm:false
        })
        for (const secret of ['email', 'gender', 'verified_at', 'password_hash', 'created_at']) {
            expect(hcr).not.toHaveProperty(secret)
        }
    })
    test('anon wants see faces 401', async () => {
        const hiker  = await asUser(BASE, {experience:"intermediate", bio:"cold brew", home_area:"serik"})
        const anon =  await request.newContext()
        await unwrap<PublicProfileResponse>(getHiker(anon, hiker.id), 401)
    })
    test('hiker does not exist', async ()=>{
        const viewer = await asUser(BASE)
        await unwrap<PublicProfileResponse>(getHiker(viewer.ctx, randomUUID()), 404)
    })
    test('counts cover past outings only, and never cancelled ones', async () => {
        const host = await asUser(BASE)
        const hiker = await asUser(BASE)
        const viewer = await asUser(BASE)
        const dayAgo = new Date(Date.now() - 24 * 3600 * 1000)
        const twoDaysAgo = new Date(Date.now() - 48 * 3600 * 1000)

        // past, attended → counts for both
        const past = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const jr = await unwrap<JoinRequestResponse>(requestJoin(hiker.ctx, past.id), 201)
        await unwrap(accept(host.ctx, jr.id), 200)
        await setOutingTimes(past.id, twoDaysAgo, dayAgo)

        // upcoming, attended → counts for nobody yet
        const upcoming = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        const jr2 = await unwrap<JoinRequestResponse>(requestJoin(hiker.ctx, upcoming.id), 201)
        await unwrap(accept(host.ctx, jr2.id), 200)

        // past but cancelled → counts for nobody
        const cancelled = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
        await unwrap(cancelOuting(host.ctx, cancelled.id), 200)
        await setOutingTimes(cancelled.id, twoDaysAgo, dayAgo)

        const hostCard = await unwrap<PublicProfileResponse>(getHiker(viewer.ctx, host.id), 200)
        expect(hostCard.hosted_count).toBe(1)
        expect(hostCard.went_count).toBe(0)

        const hikerCard = await unwrap<PublicProfileResponse>(getHiker(viewer.ctx, hiker.id), 200)
        expect(hikerCard.hosted_count).toBe(0)
        expect(hikerCard.went_count).toBe(1)
    })

})