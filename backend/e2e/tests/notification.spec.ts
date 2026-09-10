// backend/e2e/tests/notifications.spec.ts
import { test, expect } from '@playwright/test'
import { asUser, BASE } from '../fixtures'
import { unwrap } from '../envelope'
import {
    createOuting, requestJoin, accept, withdraw,
    getNotifications, readNotification, readAllNotifications,
} from '../api'
import type {
    OutingResponse, JoinRequestResponse, NotificationsResponse,
} from '../types'

test.describe('bell notifications', () => {

    test('list returns rows and unread_count, newest first', async () => {
        const host = await asUser(BASE)
        const h1 = await asUser(BASE)
        const h2 = await asUser(BASE)
        const o = await unwrap<OutingResponse>(await createOuting(host.ctx, { max_size: 6 }), 201)

        await unwrap<JoinRequestResponse>(await requestJoin(h1.ctx, o.id), 201)
        const j2 = await unwrap<JoinRequestResponse>(await requestJoin(h2.ctx, o.id), 201)
        await unwrap(await accept(host.ctx, j2.id), 200)  // host's newest event = approved-side? no —
        // host receives: created(h1), created(h2). accept notifies h2, not host.
        // so host still has exactly 2 'join_request_created'. keep it simple:

        const n = await unwrap<NotificationsResponse>(await getNotifications(host.ctx), 200)
        expect(n.notifications.length).toBe(2)
        expect(n.unread_count).toBe(2)
        // newest first: both are join_request_created; assert DESC by created_at
        const [first, second] = n.notifications
        expect(new Date(first.created_at).getTime())
            .toBeGreaterThanOrEqual(new Date(second.created_at).getTime())
    })

    test('mark one read drops unread_count', async () => {
        const host = await asUser(BASE)
        const h1 = await asUser(BASE)
        const h2 = await asUser(BASE)
        const o = await unwrap<OutingResponse>(await createOuting(host.ctx, { max_size: 6 }), 201)
        await unwrap<JoinRequestResponse>(await requestJoin(h1.ctx, o.id), 201)
        await unwrap<JoinRequestResponse>(await requestJoin(h2.ctx, o.id), 201)

        let n = await unwrap<NotificationsResponse>(await getNotifications(host.ctx), 200)
        expect(n.unread_count).toBe(2)

        await unwrap(await readNotification(host.ctx, n.notifications[0].id), 200)

        n = await unwrap<NotificationsResponse>(await getNotifications(host.ctx), 200)
        expect(n.unread_count).toBe(1)
        expect(n.notifications.find(x => x.read_at !== null)).toBeTruthy()
    })

    test('mark all read zeros unread_count', async () => {
        const host = await asUser(BASE)
        const h1 = await asUser(BASE)
        const h2 = await asUser(BASE)
        const o = await unwrap<OutingResponse>(await createOuting(host.ctx, { max_size: 6 }), 201)
        await unwrap<JoinRequestResponse>(await requestJoin(h1.ctx, o.id), 201)
        await unwrap<JoinRequestResponse>(await requestJoin(h2.ctx, o.id), 201)

        await unwrap(await readAllNotifications(host.ctx), 200)

        const n = await unwrap<NotificationsResponse>(await getNotifications(host.ctx), 200)
        expect(n.unread_count).toBe(0)
        expect(n.notifications.every(x => x.read_at !== null)).toBe(true)
    })

    test('a hiker cannot mark another hiker notification read', async () => {
        const host = await asUser(BASE)
        const attacker = await asUser(BASE)
        const h1 = await asUser(BASE)
        const o = await unwrap<OutingResponse>(await createOuting(host.ctx, { max_size: 6 }), 201)
        await unwrap<JoinRequestResponse>(await requestJoin(h1.ctx, o.id), 201)

        const n = await unwrap<NotificationsResponse>(await getNotifications(host.ctx), 200)
        expect(n.unread_count).toBe(1)
        const hostNotifId = n.notifications[0].id

        // attacker tries to mark the host's notification — 200, but no-op (ownership guard)
        await unwrap(await readNotification(attacker.ctx, hostNotifId), 200)

        const after = await unwrap<NotificationsResponse>(await getNotifications(host.ctx), 200)
        expect(after.unread_count).toBe(1)  // untouched — the guard held
    })

    test('pagination: limit and offset', async () => {
        const host = await asUser(BASE)
        const h1 = await asUser(BASE)
        const h2 = await asUser(BASE)
        const o = await unwrap<OutingResponse>(await createOuting(host.ctx, { max_size: 6 }), 201)
        await unwrap<JoinRequestResponse>(await requestJoin(h1.ctx, o.id), 201)
        await unwrap<JoinRequestResponse>(await requestJoin(h2.ctx, o.id), 201)

        // limit=1 → one row; offset=1 → the older one
        const page1 = await unwrap<NotificationsResponse>(await host.ctx.get('/api/notifications?limit=1&offset=0'), 200)  // CHECK: or a getNotifications(ctx, {limit,offset}) helper
        expect(page1.notifications.length).toBe(1)
        const page2 = await unwrap<NotificationsResponse>(await host.ctx.get('/api/notifications?limit=1&offset=1'), 200)
        expect(page2.notifications.length).toBe(1)
        expect(page1.notifications[0].id).not.toBe(page2.notifications[0].id)
    })
})