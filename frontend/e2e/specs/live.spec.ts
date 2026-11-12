// frontend/e2e/specs/live.spec.ts
import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest, withdraw} from "../fixtures/outingApiHelper.ts";
import {moveToFront} from "../fixtures/db.ts";
import {showOnBrowse} from "../fixtures/browse.ts";


// outing.changed (backend/docs/events/outing-changed.md): a signed-in tab
// refetches when any outing changes, with no reload and no focus event.
// The change is made through the API from a different actor, so the page
// under test can only learn about it from the stream.
test.describe('outing.changed keeps lists live', () => {
    const cardFor = (page: import('@playwright/test').Page, title: string) =>
        page.locator('a', {has: page.getByRole('heading', {name: title})})

    test('browse card moves when a member withdraws elsewhere', async ({page, context}) => {
        const host = await createActor()
        const member = await createActor()
        const viewer = await createActor()   // a stranger to the outing — fan-out to the audience would miss them

        const outing = await createOuting(host, {max_size: 8, host_seats: 4})
        const jr = await joinRequest(member, outing.id, {guests: 1})
        await acceptRequest(host, jr.id)

        await actorInBrowser(viewer, context)
        await moveToFront(outing.id)
        await page.goto('/')
        await showOnBrowse(page, outing.title)
        const card = cardFor(page, outing.title)
        await expect(card.getByText('3 of 8 going · 1 spot left', {exact: true})).toBeVisible()

        await withdraw(member, outing.id)

        await expect(card.getByText('1 of 8 going · 3 spots left', {exact: true})).toBeVisible()
    })

    test('my outings card moves when a request is accepted elsewhere', async ({page, context}) => {
        const host = await createActor()
        const rider = await createActor()

        const outing = await createOuting(host, {max_size: 8, host_seats: 4})
        const jr = await joinRequest(rider, outing.id)

        await actorInBrowser(host, context)
        await page.goto('/me/outings')
        const card = cardFor(page, outing.title)
        await expect(card.getByText('1 of 8 going · 3 spots left · 1 waiting', {exact: true})).toBeVisible()

        await acceptRequest(host, jr.id)

        await expect(card.getByText('2 of 8 going · 2 spots left', {exact: true})).toBeVisible()
    })
})