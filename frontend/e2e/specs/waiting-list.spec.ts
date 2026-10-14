import {expect, test} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, declineRequest, joinRequest, withdraw} from "../fixtures/outingApiHelper.ts";


test.describe("waiting and withdraw list", ()=>{
    test("a logged-in stranger sees both lists, in order", async ({page, context})=>{
        const host = await createActor()
        const hikerA = await createActor()
        const hikerB = await  createActor()
        const hikerC = await createActor()
        const stranger = await createActor()

        const  o = await createOuting(host)
        await joinRequest(hikerA, o.id, {guests:2})
        await joinRequest(hikerB, o.id)
        await joinRequest(hikerC, o.id)
        await withdraw(hikerC, o.id)

        await actorInBrowser(stranger, context)
        await page.goto(`/outings/${o.id}`)
        await expect(page.getByRole('heading', {name: 'Waiting (2)'})).toBeVisible()

        // waiting: the two chips, in queue order
        const waiting = page.getByRole('list').filter({hasText: hikerA.user.name}).getByRole('listitem')
        await expect(waiting).toContainText([hikerA.user.name, hikerB.user.name])

        // withdrew
        await expect(page.getByRole('heading', {name: 'Withdrew (1)'})).toBeVisible()
        await expect(page.getByText(hikerC.user.name, {exact: true})).toBeVisible()
    })
    test('a logged-out visitor sees a count, not names', async ({page}) => {
        const host = await createActor()
        const hikerA = await createActor()
        const hikerB = await createActor()
        const hikerC = await createActor()

        const o = await createOuting(host, {max_size:8, host_seats:4})
        const jrA = await joinRequest(hikerA, o.id, {guests: 2})
        await acceptRequest(host, jrA.id)          // A is going, with 2 guests
        await joinRequest(hikerB, o.id)            // B is waiting
        await joinRequest(hikerC, o.id)
        await withdraw(hikerC, o.id)               // C withdrew

        await page.goto(`/outings/${o.id}`)

        // the count: A + 2 guests = 3 people besides the host
        const more = page.getByText('3 more going.')
        await expect(more).toBeVisible()
        await expect(more.getByRole('link', {name: 'Log in'})).toBeVisible()

        // no names, no lists
        await expect(page.getByText(hikerA.user.name)).toHaveCount(0)
        await expect(page.getByText(hikerB.user.name)).toHaveCount(0)
        await expect(page.getByText(hikerC.user.name)).toHaveCount(0)
        await expect(page.getByRole('heading', {name: /Waiting/})).toHaveCount(0)
        await expect(page.getByRole('heading', {name: /Withdrew/})).toHaveCount(0)
    })
    test("host restores a declined request to the front of the waiting list", async ({page, context}) => {
        const host = await createActor()
        const hikerA = await createActor()
        const hikerB = await createActor()
        const o = await createOuting(host, {max_size: 8, host_seats: 4})
        const jrA = await joinRequest(hikerA, o.id, {guests: 2})
        await joinRequest(hikerB, o.id)
        await declineRequest(host, jrA.id)

        await actorInBrowser(host, context)
        await page.goto(`/outings/${o.id}`)

        const pendingRows = page.getByRole('button', { name: /requests as/ })
        const tab = (name: string) => page.getByRole('tab', { name })

        // only B is waiting; A sits in Declined with a Restore button
        await expect(pendingRows).toContainText([hikerB.user.name])
        await tab('Declined').click()
        const restore = page.getByRole('button', { name: 'Restore' })
        await expect(restore).toBeVisible()

        // restore: Declined empties, A is back in front of B
        await restore.click()
        await expect(page.getByText("You haven't declined anyone.")).toBeVisible()
        await expect(tab('Waiting')).toContainText('2')
        await tab('Waiting').click()
        await expect(pendingRows).toContainText([hikerA.user.name, hikerB.user.name])
    })
})