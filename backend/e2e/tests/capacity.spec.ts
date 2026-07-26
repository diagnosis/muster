import {expect, test} from '@playwright/test'
import {asUser} from "../fixtures";
import {accept, createOuting, requestJoin} from "../api";
import {JoinRequestResponse, OutingResponse} from "../types";

const BASE = "http://localhost:8088"

test.describe('capacity', ()=> {
    test(`
    create outing (max_size:2, host_seat:4, starts_at: 48h) -> 
    rider with a guest request to join the outing -> bounces 409
    
    `,  async () => {
        const {ctx:ctxHost, user:host} = await asUser(BASE)
        let res = await createOuting(ctxHost, {max_size:2, host_seats:4})
        expect(res.status()).toBe(201)
        const outing:OutingResponse = (await res.json())['data']
        expect(outing.id).toBeTruthy()

        const {ctx:ctxHiker, user:hiker} = await asUser(BASE)

        res = await requestJoin(ctxHiker, outing.id, {guests:1})

        expect(res.status()).toBe(201)

        const joinRequestResponse: JoinRequestResponse = (await res.json())['data']
        expect(joinRequestResponse.status).toBe('requested')

        res = await accept(ctxHost,joinRequestResponse.id)
        expect(res.status()).toBe(409)

    })

    test(`
    create outing (max_size:2, host_seat:4, starts_at: 48h) -> 
    single rider requests to join the outing -> 200
    `, async ()=> {
        const {ctx:ctxHost, user:host} = await asUser(BASE)
        let res = await createOuting(ctxHost, {max_size:2, host_seats:4})
        expect(res.status()).toBe(201)
        const outing:OutingResponse = (await res.json())['data']
        expect(outing.id).toBeTruthy()

        const {ctx:ctxHiker, user:hiker} = await asUser(BASE)

        res = await requestJoin(ctxHiker, outing.id, {guests:0})

        expect(res.status()).toBe(201)

        const joinRequestResponse: JoinRequestResponse = (await res.json())['data']
        expect(joinRequestResponse.status).toBe('requested')

        res = await accept(ctxHost,joinRequestResponse.id)
        expect(res.status()).toBe(200)

        res = await ctxHiker.get(`/api/outings/${outing.id}`);
        const detail = (await res.json()).data;
        expect(detail.my_request.status).toBe('accepted');
        expect(detail.people_count).toBe(2);   // the hand-math, asserted live
        expect(detail.spots_left).toBe(0);
    })

    test('driver with surplus seats bounced at FULL outing', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        let res = await createOuting(ctxHost, { max_size: 2, host_seats: 4 });
        const outing = (await res.json()).data;

        // FILL IT: rider in, outing now 2/2
        const { ctx: ctxRider } = await asUser(BASE);
        res = await requestJoin(ctxRider, outing.id);
        const riderReq = (await res.json()).data;
        res = await accept(ctxHost, riderReq.id);
        expect(res.status()).toBe(200);

        // NOW the driver — full outing, surplus seats, no guest
        const { ctx: ctxDriver } = await asUser(BASE);
        res = await requestJoin(ctxDriver, outing.id, { role: 'driver', seats_offered: 4 });
        expect(res.status()).toBe(201);
        const driverReq = (await res.json()).data;

        res = await accept(ctxHost, driverReq.id);
        expect(res.status()).toBe(409);   // cap says no, seats don't matter
    });

    test('driver with no guest accepted to outing', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        let res = await createOuting(ctxHost, { max_size: 2, host_seats: 4 });
        const outing = (await res.json()).data;

        const { ctx: ctxDriver } = await asUser(BASE);
        res = await requestJoin(ctxDriver, outing.id, { role: 'driver', seats_offered: 4 });
        expect(res.status()).toBe(201);
        const driverReq = (await res.json()).data;

        res = await accept(ctxHost, driverReq.id);
        expect(res.status()).toBe(200);
    });

    test('guests consumes slots: outing max 4 rider + 2 guests accepted', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        let res = await createOuting(ctxHost, { max_size: 4, host_seats: 4 });
        const outing = (await res.json()).data;

        const { ctx: ctxRider } = await asUser(BASE);
        res = await requestJoin(ctxRider, outing.id, { role: 'rider', guests:2 });
        expect(res.status()).toBe(201);
        const riderReq = (await res.json()).data;

        res = await accept(ctxHost, riderReq.id);
        expect(res.status()).toBe(200);

        res = await ctxRider.get(`/api/outings/${outing.id}`);
        const detail = (await res.json()).data;
        expect(detail.my_request.status).toBe('accepted');
        expect(detail.people_count).toBe(4);
        expect(detail.spots_left).toBe(0);

    });

    test('Capacity zero host offers no seat', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        let res = await createOuting(ctxHost, { max_size: 4, host_seats: 0 });
        const outing = (await res.json()).data;

        const { ctx: ctxRider } = await asUser(BASE);
        res = await requestJoin(ctxRider, outing.id, { role: 'rider', guests:0 });
        expect(res.status()).toBe(201);
        const riderReq = (await res.json()).data;

        res = await accept(ctxHost, riderReq.id);
        expect(res.status()).toBe(409);

    });






})