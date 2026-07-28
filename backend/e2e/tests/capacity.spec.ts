import {expect, test} from '@playwright/test'
import {asUser,BASE} from "../fixtures";
import {accept, createOuting, getDetail, requestJoin} from "../api";
import {DetailResponse, JoinRequestResponse, OutingResponse} from "../types";
import {unwrap} from "../envelope";


test.describe('capacity', ()=> {
    test(`
    create outing (max_size:2, host_seat:4, starts_at: 48h) -> 
    rider with guest bounced at empty-but-tight outing (need 3 > max 2)
    
    `,  async () => {
        const {ctx:ctxHost} = await asUser(BASE)

        const outing =
            await unwrap<OutingResponse>(createOuting(ctxHost, {max_size:2, host_seats:4}), 201)
        const {ctx:ctxHiker} = await asUser(BASE)

        const joinRequestResponse =
            await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id, {guests:1}), 201)
        expect(joinRequestResponse.status).toBe('requested')

        const res = await accept(ctxHost, joinRequestResponse.id);
        expect(res.status()).toBe(409);
    })

    test(`
    create outing (max_size:2, host_seat:4, starts_at: 48h) -> 
    single rider requests to join the outing -> 200
    `, async ()=> {
        const {ctx:ctxHost} = await asUser(BASE)
        const outing =
            await unwrap<OutingResponse>(createOuting(ctxHost, {max_size:2, host_seats:4}), 201)

        const {ctx:ctxHiker} = await asUser(BASE)

        const joinRequestResponse =
            await unwrap<JoinRequestResponse>(requestJoin(ctxHiker, outing.id, {guests:0}), 201)
        expect(joinRequestResponse.status).toBe('requested')


        const res = await accept(ctxHost,joinRequestResponse.id)
        expect(res.status()).toBe(200)


        const detail = await unwrap<DetailResponse>(getDetail(ctxHiker, outing.id),200)
        expect(detail.my_request?.status).toBe('accepted');
        expect(detail.people_count).toBe(2);   // the hand-math, asserted live
        expect(detail.spots_left).toBe(0);
    })

    test('driver with surplus seats bounced at FULL outing', async () => {
        const { ctx: ctxHost } = await asUser(BASE);

        const outing = await unwrap<OutingResponse>(createOuting(ctxHost, {max_size:2, host_seats:4}), 201)

        // FILL IT: rider in, outing now 2/2
        const { ctx: ctxRider } = await asUser(BASE);
        const riderReq = await unwrap<JoinRequestResponse>(requestJoin(ctxRider, outing.id), 201)

        let res = await accept(ctxHost, riderReq.id);
        expect(res.status()).toBe(200);

        // NOW the driver — full outing, surplus seats, no guest
        const { ctx: ctxDriver } = await asUser(BASE);
        const driverReq = await unwrap<JoinRequestResponse>(requestJoin(ctxDriver, outing.id, {role: 'driver', seats_offered:4}), 201)

        res = await accept(ctxHost, driverReq.id);
        expect(res.status()).toBe(409);   // cap says no, seats don't matter
    });

    test('driver with no guest accepted to outing', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost, {max_size:2, host_seats:4}), 201)


        const { ctx: ctxDriver } = await asUser(BASE);
        const driverReq = await unwrap<JoinRequestResponse>(requestJoin(ctxDriver, outing.id, {role: 'driver', seats_offered:4}), 201)

        const res = await accept(ctxHost, driverReq.id);
        expect(res.status()).toBe(200);
    });

    test('guests consumes slots: outing max 4 rider + 2 guests accepted', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost, {max_size: 4, host_seats: 4}), 201)

        const { ctx: ctxRider } = await asUser(BASE);
        const riderReq = await unwrap<JoinRequestResponse>(requestJoin(ctxRider, outing.id, {role:'rider', guests:2}), 201)

        const res = await accept(ctxHost, riderReq.id);
        expect(res.status()).toBe(200);

        const detail = await unwrap<DetailResponse>(getDetail(ctxRider, outing.id), 200)

        expect(detail.my_request?.status).toBe('accepted');
        expect(detail.people_count).toBe(4);
        expect(detail.spots_left).toBe(0);

    });

    test('Capacity zero host offers no seat', async () => {
        const { ctx: ctxHost } = await asUser(BASE);
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost, {max_size:4, host_seats:0}), 201)


        const { ctx: ctxRider } = await asUser(BASE);
        const riderReq = await unwrap<JoinRequestResponse>(requestJoin(ctxRider, outing.id, {role:'rider', guests:0}), 201)

        const res = await accept(ctxHost, riderReq.id);
        expect(res.status()).toBe(409);

    });






})