import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest, removeMember} from "../fixtures/outingApiHelper.ts";
import {REMOVE_BTN_TEXT, REMOVED} from "../fixtures/copy.ts";

test.describe("member removal from outing", ()=>{
    test('host should successfully remove any member from the roster', async ({page, context})=>{
        const host = await createActor()
        const hiker1 = await createActor()
        const hiker2 = await createActor()
        const outing = await createOuting(host, {max_size:6, host_seats:4})
        const jr = await joinRequest(hiker1, outing.id, {guests:1})
        const jr2 = await joinRequest(hiker2, outing.id, {guests:0})
        await acceptRequest(host, jr.id)
        await acceptRequest(host, jr2.id)
        await actorInBrowser(host, context)
        await page.goto(`/outings/${outing.id}`)
        //deleting hiker 1
        const hiker1Row = page.getByText(`${hiker1.user.name} · ${hiker1.user.experience} `)
        const hiker2Row = page.getByText(`${hiker2.user.name} · ${hiker2.user.experience}`)
        const hiker1RemoveBtn = hiker1Row.getByRole('button', { name: 'Remove' })
        await hiker1RemoveBtn.click()
        //modal opens
        const modalTitle = `Remove ${hiker1.user.name}`
        const modal = page.getByRole('dialog', {name:modalTitle})

        await expect(modal.getByRole('heading', {name:modalTitle})).toBeVisible()
        await page.getByRole('button', { name: REMOVE_BTN_TEXT }).click()
        await expect(modal).not.toBeVisible()
        await expect(hiker1Row).not.toBeVisible()
        await expect(hiker2Row).toBeVisible()
    });
    test('removed member should see proper message', async ({page, context})=>{
       const host  = await createActor()
       const hiker = await createActor()
       const outing = await createOuting(host, {max_size:4, host_seats:4})
       const jr = await joinRequest(hiker, outing.id)
       await acceptRequest(host, jr.id)
       await removeMember(host, jr.id)

       await actorInBrowser(hiker, context)
       await page.goto(`/outings/${outing.id}`)
       await expect(page.getByText(REMOVED)).toBeVisible()
    });
})