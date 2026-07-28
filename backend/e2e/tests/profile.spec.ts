import {expect, request, test} from '@playwright/test'
import {asUser} from "../fixtures";
import {createOuting, getDetail, getHiker, getMe, updateProfile} from "../api";
import {DetailResponse, MemberCard, MeResponse, OutingResponse} from "../types";
import {unwrap} from "../envelope";

const BASE = "http://localhost:8088"
test.describe("profile", ()=>{
    test('profile patch propagates to me and detail host card', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

        // patch my name
        await unwrap<MeResponse>(updateProfile(ctxHost, {name:'Ziplayan Ceylan'}),200)
        // /me shows it
        const me = await unwrap<MeResponse>(getMe(ctxHost), 200)
        expect(me.name).toBe('Ziplayan Ceylan');

        // detail's host card shows it — the JOIN carrying the edit
        const detail = await unwrap<DetailResponse>(getDetail(ctxHost, outing.id), 200)
        expect(detail.host.name).toBe('Ziplayan Ceylan');
    });

    test('anonymous hiker card serves id/name/experience, no email', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

        const ctxAnon = await request.newContext({ baseURL: BASE });

        const card = await unwrap<MemberCard>(getHiker(ctxAnon, outing.host_id), 200)

        expect(card.id).toBe(outing.host_id);
        expect(card.name).toBe('Test User');
        expect(card.experience).toBe('beginner');
        expect(card.email).toBeUndefined();       // the privacy assertion — the test's point
    });

    test('invalid experience -> 400', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        const res = await updateProfile(ctxHost, {experience:'expert'})
        expect(res.status()).toBe(400);
    });

})