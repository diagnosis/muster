

import {expect, request, test} from '@playwright/test'
import {asUser, BASE} from "../fixtures";
import {createOuting, listOutings} from "../api";
import {OutingResponse, PageResponse} from "../types";
import {unwrap} from "../envelope";
import {setOutingTimes} from "../db";

test.describe('pagination', () => {
    test('following next_cursor reaches every outing exactly once, across a tie', async () => {
        const host = await asUser(BASE)

        // Three outings at the same instant, a few minutes from now: they sort to
        // the front of the list, and with limit 2 a tied pair straddles a page
        // boundary, so the (starts_at, id) tiebreak runs in the real query.
        const soon = new Date(Date.now() + 5 * 60 * 1000)
        const mine: string[] = []
        for (let i = 0; i < 3; i++) {
            const o = await unwrap<OutingResponse>(createOuting(host.ctx), 201)
            await setOutingTimes(o.id, soon, null)
            mine.push(o.id)
        }

        const anon = await request.newContext({baseURL: BASE})
        const seen: string[] = []
        let after: string | undefined

        // Stop when all three have turned up, or the list ends, or after ten pages.
        for (let pageNo = 0; pageNo < 10; pageNo++) {
            const page = await unwrap<PageResponse>(listOutings(anon, {limit: 2, after}), 200)
            expect(page.items.length).toBeLessThanOrEqual(2)
            for (const o of page.items) {
                seen.push(o.id)
            }

            if (mine.every(id => seen.includes(id))) break
            if (page.next_cursor === null) break
            after = page.next_cursor
        }

        // nothing skipped: each of my outings appeared
        for (const id of mine) {
            expect(seen, `outing ${id} never appeared on any page`).toContain(id)
        }
        // nothing repeated: no id came back on two pages
        expect(new Set(seen).size, 'an outing appeared on more than one page').toBe(seen.length)
    })

    test('a cursor the server did not issue is a 400', async () => {
        const anon = await request.newContext({baseURL: BASE})
        const res = await listOutings(anon, {limit: 20, after: 'not-a-cursor'})
        expect(res.status()).toBe(400)
    })

    test('limit is clamped to the maximum page size, never refused', async () => {
        const anon = await request.newContext({baseURL: BASE})
        const page = await unwrap<PageResponse>(listOutings(anon, {limit: 1000}), 200)
        expect(page.items.length).toBeLessThanOrEqual(50)
    })
})