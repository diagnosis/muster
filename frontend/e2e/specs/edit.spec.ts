import {expect, test} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest, uniqueOuting} from "../fixtures/outingApiHelper.ts";
import {MAX_SIZE_SHRINK_CONFLICT} from "../fixtures/copy.ts";


test.describe("edit", ()=> {
    test('host edits title, details updates without reload', async ({page, context})=>{
        const host = await createActor()
        const outing = await createOuting(host)

        await actorInBrowser(host, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByRole('heading', { name: outing.title})).toBeVisible()
        await page.getByRole('link', { name: 'Edit outing' }).click()

        await expect(page.getByRole('textbox', {name:'Title'})).toHaveValue(outing.title)
        await expect(page.getByRole('textbox', { name: 'Destination' })).toHaveValue(outing.destination)
        await expect(page.getByRole('textbox', { name: 'Meet label' })).toHaveValue(outing.meet_label)


        const newOuting = uniqueOuting()
        await page.getByRole('textbox', { name: 'Title' }).fill(newOuting.title)
        await page.getByRole('button', { name: 'Save changes' }).click()
        await expect(page.getByRole('heading', { name: newOuting.title })).toBeVisible()
    });
    test('non-host is bounced from edit', async ({page, context})=>{
        const host = await createActor()
        const outsider = await createActor()
        const outing = await createOuting(host)

        await actorInBrowser(outsider, context)
        await page.goto(`/outings/${outing.id}/edit`)

        await expect(page).toHaveURL(/\/outings\/[0-9a-f-]+$/)

    });
    test('shrink below committed members renders conflict', async({page, context})=>{
        const host = await createActor()
        const outing = await createOuting(host, {max_size:6, host_seats:4})
        const hiker1 = await createActor()
        const joinReq = await joinRequest(hiker1,outing.id, {guests:2, role:'rider'})
        await acceptRequest(host, joinReq.id)

        await actorInBrowser(host, context)
        await page.goto(`/outings/${outing.id}/edit`)
        await page.getByRole('spinbutton', { name: 'Max size' }).fill('3')
        await page.getByRole('button', {name:'Save changes'}).click()
        await expect(page.getByText(MAX_SIZE_SHRINK_CONFLICT)).toBeVisible()
    })
})