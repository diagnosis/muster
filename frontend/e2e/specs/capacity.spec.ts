import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest, updateOuting} from "../fixtures/outingApiHelper.ts";
import {FULL_WARNING, OUTING_FULL, SEAT_SHORT, NO_SEAT_LEFT} from "../fixtures/copy.ts";


test.describe('outing capacity', ()=>{
    test('cap-bound-full', async ({page, context})=> {
        const actorHost = await createActor()
        const outing = await createOuting(actorHost, {max_size:2, host_seats:2})
        const actorHiker = await createActor()
        const jr = await joinRequest(actorHiker, outing.id)
        await acceptRequest(actorHost, jr.id)
        await page.goto(`/outings/${outing.id}`)

        await expect(page.getByText(OUTING_FULL)).toBeVisible()
        const actorStranger = await createActor()
        await actorInBrowser(actorStranger, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByText(FULL_WARNING)).toBeVisible()
    });
    test('seat-bound face', async ({page,context})=>{
        const actorHost = await createActor()
        const actorHiker = await createActor()
        const actorDriver = await createActor()

        const outing = await createOuting(actorHost, {max_size:8, host_seats:2})
        const jr = await joinRequest(actorHiker, outing.id)
        await acceptRequest(actorHost, jr.id)
        await actorInBrowser(actorDriver, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByText(NO_SEAT_LEFT)).toBeVisible()
        await expect(page.getByText(OUTING_FULL)).not.toBeVisible()
        await expect(page.getByText(FULL_WARNING)).not.toBeVisible()
     });
    test('host_seats:0', async ({page, context})=>{
        const actorHost = await createActor()
        const outing = await createOuting(actorHost, {host_seats:0, max_size:4})
        const actorHiker = await createActor()
        await actorInBrowser(actorHiker, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByText(SEAT_SHORT)).toBeVisible()
        await expect(page.getByText(NO_SEAT_LEFT)).not.toBeVisible()
        await expect(page.getByText(OUTING_FULL)).not.toBeVisible()

    })
    test("browse cards say how much room is left, in the right words", async ({page, context}) => {
        const host = await createActor()
        const rider = await createActor()
        const driver = await createActor()
        const waiter = await createActor()

        // short: accept at 3 seats, then the host shrinks to 2 → 3 people in 2 seats, 1 short.
        // (Accept refuses past capacity, so shortage only happens after the fact.)
        const short = await createOuting(host, {max_size: 8, host_seats: 3})
        const j1 = await joinRequest(rider, short.id, {guests: 1})
        await acceptRequest(host, j1.id)
        await updateOuting(host, short.id, {host_seats: 2})

        // no seats left under max: host seats 1 + driver's 1 = 2 seats for 2 people, max 8, one waiting
        const seatless = await createOuting(host, {max_size: 8, host_seats: 1})
        const j2 = await joinRequest(driver, seatless.id, {role: 'driver', seats_offered: 1})
        await acceptRequest(host, j2.id)
        await joinRequest(waiter, seatless.id)

        await actorInBrowser(rider, context)
        await page.goto('/')

        const cardFor = (title: string) => page.locator('a', {has: page.getByRole('heading', {name: title})})
        await expect(cardFor(short.title).getByText('3 of 8 going · 1 seat short, a driver would help', {exact: true})).toBeVisible()
        await expect(cardFor(seatless.title).getByText('2 of 8 going · no seats left, a driver would open more · 1 waiting', {exact: true})).toBeVisible()
    })
})
