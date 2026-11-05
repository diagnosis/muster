// frontend/e2e/specs/hiker-card.spec.ts
import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest} from "../fixtures/outingApiHelper.ts";

test.describe('hiker card', () => {
    test("a roster name opens that hiker's card", async ({page, context}) => {
        const host = await createActor()
        const hiker = await createActor()
        const outing = await createOuting(host, {max_size: 6, host_seats: 4})
        const jr = await joinRequest(hiker, outing.id)
        await acceptRequest(host, jr.id)

        await actorInBrowser(hiker, context)
        await page.goto(`/outings/${outing.id}`)
        await page.getByRole('link', {name: host.user.name}).click()

        await expect(page).toHaveURL(`/hikers/${host.hikerID}`)
        await expect(page.getByRole('heading', {level: 1, name: host.user.name})).toBeVisible()
        await expect(page.getByText('Hosted')).toBeVisible()
        // the hiker shares an outing with the host, so the server says they may message
        await expect(page.getByRole('button', {name: 'Message'})).toBeVisible()
        await expect(page.getByRole('link', {name: 'Edit profile'})).toHaveCount(0)
    })

    test('what I write on my profile shows on my own card', async ({page, context}) => {
        const me = await createActor()
        await actorInBrowser(me, context)

        await page.goto('/me/profile')
        await page.getByRole('textbox', {name: 'Home area'}).fill('Renton')
        await page.getByRole('textbox', {name: 'About you'}).fill('Cold brew and switchbacks.')
        await page.getByRole('button', {name: 'Save changes'}).click()
        await expect(page.getByText('Nothing to save yet.')).toBeVisible()   // dirty flag cleared = saved

        await page.goto(`/hikers/${me.hikerID}`)
        await expect(page.getByRole('heading', {level: 1, name: me.user.name})).toBeVisible()
        await expect(page.getByText('Beginner · Renton')).toBeVisible()
        await expect(page.getByText('Cold brew and switchbacks.')).toBeVisible()
        await expect(page.getByRole('link', {name: 'Edit profile'})).toBeVisible()
        await expect(page.getByRole('button', {name: 'Message'})).toHaveCount(0)
    })
})