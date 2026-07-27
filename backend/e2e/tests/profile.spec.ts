import {expect, request, test} from '@playwright/test'
import {asUser} from "../fixtures";
import {createOuting, getDetail} from "../api";
import {DetailResponse, OutingResponse} from "../types";

const BASE = "http://localhost:8088"
test.describe("profile", ()=>{
    test('profile patch propagates to me and detail host card', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        let res = await createOuting(ctxHost);
        expect(res.status()).toBe(201);
        const outingData: OutingResponse = (await res.json()).data;

        // patch my name
        res = await ctxHost.patch('/api/me/profile', { data: { name: 'Ziplyan II' } });
        expect(res.status()).toBe(200);

        // /me shows it
        res = await ctxHost.get('/api/auth/me');
        expect(res.status()).toBe(200);
        expect((await res.json()).data.name).toBe('Ziplyan II');

        // detail's host card shows it — the JOIN carrying the edit
        res = await getDetail(ctxHost, outingData.id);
        expect(res.status()).toBe(200);
        const detail: DetailResponse = (await res.json()).data;
        expect(detail.host.name).toBe('Ziplyan II');
    });

    test('anonymous hiker card serves id/name/experience, no email', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        let res = await createOuting(ctxHost);
        const outingData: OutingResponse = (await res.json()).data;

        const ctxAnon = await request.newContext({ baseURL: BASE });
        res = await ctxAnon.get(`/api/hikers/${outingData.host_id}`);
        expect(res.status()).toBe(200);
        const card = (await res.json()).data;

        expect(card.id).toBe(outingData.host_id);
        expect(card.name).toBe('Test User');
        expect(card.experience).toBe('beginner');
        expect(card.email).toBeUndefined();       // the privacy assertion — the test's point
    });

    test('invalid experience -> 400', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        const res = await ctxHost.patch('/api/me/profile', { data: { experience: 'expert' as any } });
        expect(res.status()).toBe(400);
    });

})