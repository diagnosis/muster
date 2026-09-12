import {expect, test} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {createOuting, joinRequest} from "../fixtures/outingApiHelper.ts";

test.describe("notification bell", ()=>{
    test('receive, view and clear notification', async ({page, context})=>{
        const host = await createActor()
        const hiker = await createActor()
        const o = await createOuting(host, {max_size:8, host_seats:4})
        await joinRequest(hiker, o.id, {role:"rider", guests:2, note:"Halo!"})

        await actorInBrowser(host, context)
        await page.goto("/")
        await expect(page.getByRole('button', {name:'Notifications'})).toContainText('1')
        await page.getByRole('button', {name:'Notifications'}).click()
        await expect(page.getByText(`New request to join ${o.title}`)).toBeVisible()
        await page.getByRole('button', { name: 'Mark all read' }).click()
        await expect(page.getByRole('button', {name:'Notifications'})).not.toContainText('1')
    })
})