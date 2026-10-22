import {expect, test} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {
    acceptRequest,
    createOuting,
    declineRequest,
    joinRequest,
    listOutingRequests,
    withdraw
} from "../fixtures/outingApiHelper.ts";



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

        const summary = page.getByText(`${hikerA.user.name} and ${hikerB.user.name}`, {exact: true})
        await expect(summary).toBeVisible()

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
    test("hiker edits their request and move to back", async ({page, context})=>{
        const host = await createActor()
        const hiker1 = await createActor()
        const hiker2 = await createActor()
        const driver = await createActor()
        const o = await createOuting(host, {host_seats:3, max_size:8})
        const j1 = await joinRequest(hiker1, o.id, {role:"rider", guests:1})
        const j2 = await joinRequest(hiker2, o.id)
        const j3 = await joinRequest(driver, o.id, {role:"driver", seats_offered:2})

        await actorInBrowser(hiker1, context)
        await page.goto(`/outings/${o.id}`)
        await expect(page.getByText("You asked as a rider, bringing 1 guest", {exact:true})).toBeVisible()
        await expect(page.getByText("You're 2nd of 3", {exact: false})).toBeVisible()
        await page.getByRole('button', { name: 'Edit request' }).click()
        await page.getByRole('spinbutton', { name: 'Guests' }).fill("2")
        await expect(page.getByText('Saving this moves you to the back of the waiting list.')).toBeVisible()
        await page.getByRole('button', { name: 'Save changes' }).click()
        await expect(page.getByText("You asked as a rider, bringing 2 guests", {exact:true})).toBeVisible()

        await expect(page.getByText("You're 3rd of 3", {exact: false})).toBeVisible()
        const jrws = await listOutingRequests(host, o.id, "requested")
        const ids = jrws.map(jrw => jrw.id)
        expect(ids).toEqual([j3.id,j2.id,j1.id])

    })
    test("waiting hiker sees the full queue with their place", async ({page, context}) => {
        const host = await createActor()
        const driver = await createActor()
        const r1 = await createActor()
        const me = await createActor()
        const r3 = await createActor()
        const o = await createOuting(host, {host_seats: 1, max_size: 8})   // nobody gets accepted
        await joinRequest(r1, o.id)
        await joinRequest(me, o.id, {guests: 1})
        await joinRequest(r3, o.id)
        await joinRequest(driver, o.id, {role: 'driver', seats_offered: 2})   // asks last, leads anyway

        await actorInBrowser(me, context)
        await page.goto(`/outings/${o.id}`)

        // summary: three names and the rest as a count
        await expect(page.getByRole('heading', {name: 'Waiting (4)'})).toBeVisible()
        await expect(page.getByText('and 1 more')).toBeVisible()
        await expect(page.getByRole('status')).toContainText("You're 3rd of 4")

        // the modal: ordered, tagged, me marked
        await page.getByRole('button', {name: 'See all'}).click()
        const dialog = page.getByRole('dialog', {name: 'Waiting (4)'})
        await expect(dialog).toBeVisible()
        const rows = dialog.getByRole('listitem')
        await expect(rows).toHaveCount(4)
        await expect(rows.nth(0)).toContainText(driver.user.name)
        await expect(rows.nth(0)).toContainText('Driver · 2 seats')
        await expect(rows.nth(2)).toContainText(me.user.name)
        await expect(rows.nth(2)).toContainText('(you)')
        await expect(rows.nth(2)).toContainText('+1 guest')
        await expect(rows.nth(2)).toHaveAttribute('aria-current', 'true')

        await dialog.getByRole('button', {name: 'Close'}).click()
        await expect(dialog).toHaveCount(0)
    })

})