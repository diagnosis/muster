// backend/e2e/tests/my-outings.spec.ts
import {expect, test} from '@playwright/test'

import {accept, createOuting, decline, listForMe, requestJoin, withdraw} from '../api'
import type {JoinRequestResponse, MyOutingsResponse, OutingResponse} from '../types'
import {asUser, BASE} from "../fixtures";
import {unwrap} from "../envelope";

// GET /api/me/outings buckets by the caller's relationship to each outing:
// hosting, joined (accepted), requested (pending). Declined and withdrawn
// requests put the outing in none of them. Every bucket carries capacity.
test.describe('my outings', () => {
    test('hosting, joined and requested are exact; declined and withdrawn appear nowhere', async () => {
        const hostA = await asUser(BASE)
        const hostB = await asUser(BASE)
        const hikerA = await asUser(BASE)          // accepted → joined
        const hikerB = await asUser(BASE)          // pending  → requested
        const hikerC = await asUser(BASE)          // declined → nowhere
        const hikerD = await asUser(BASE)          // withdrawn → nowhere

        const oA = await unwrap<OutingResponse>(createOuting(hostA.ctx), 201)
        const oB = await unwrap<OutingResponse>(createOuting(hostB.ctx), 201)

        const jA = await unwrap<JoinRequestResponse>(requestJoin(hikerA.ctx, oA.id), 201)
        await unwrap(accept(hostA.ctx, jA.id), 200)
        await unwrap(requestJoin(hikerB.ctx, oA.id), 201)
        const jC = await unwrap<JoinRequestResponse>(requestJoin(hikerC.ctx, oA.id), 201)
        await unwrap(decline(hostA.ctx, jC.id), 200)
        await unwrap(requestJoin(hikerD.ctx, oA.id), 201)
        await unwrap(withdraw(hikerD.ctx, oA.id), 200)

        const ids = (list: OutingResponse[]) => list.map(o => o.id)

        const hostList = await unwrap<MyOutingsResponse>(listForMe(hostB.ctx), 200)
        expect(ids(hostList.hosting)).toEqual([oB.id])
        expect(hostList.joined).toEqual([])
        expect(hostList.requested).toEqual([])

        const joined = await unwrap<MyOutingsResponse>(listForMe(hikerA.ctx), 200)
        expect(joined.hosting).toEqual([])
        expect(ids(joined.joined)).toEqual([oA.id])
        expect(joined.requested).toEqual([])
        expect(joined.joined[0].capacity).toBeDefined()

        const requested = await unwrap<MyOutingsResponse>(listForMe(hikerB.ctx), 200)
        expect(requested.hosting).toEqual([])
        expect(requested.joined).toEqual([])
        expect(ids(requested.requested)).toEqual([oA.id])
        expect(requested.requested[0].capacity).toBeDefined()

        for (const gone of [hikerC, hikerD]) {
            const list = await unwrap<MyOutingsResponse>(listForMe(gone.ctx), 200)
            expect(list).toEqual({hosting: [], joined: [], requested: []})
        }
    })
})