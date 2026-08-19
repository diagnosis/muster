
//e2e/specs/join-form.spec.ts
import {expect, type Page, test, type BrowserContext} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {createOuting} from "../fixtures/outingApiHelper.ts";

test.describe('join form', ()=>{
    test('null-role disables submit', async ({page, context})=>{
        await openJoinForm(page, context)
        await expect(page.getByRole('button', { name: 'Request to join' })).toBeDisabled()
        await expect(page.getByRole('button', { name: 'Never mind' })).toBeEnabled()
    });
    test('seats visibility follow role', async ({page, context})=>{
        await openJoinForm(page, context)
        const roles = page.getByRole('group', { name: 'Hiker Role' })
        await roles.getByText('Rider').click()
        await expect(page.getByRole('button', {name:'Request to join'})).toBeEnabled()
        await expect(page.getByRole('spinbutton', { name: 'Seats Offered' })).not.toBeVisible()
        await roles.getByText('Driver').click()
        await expect(page.getByRole('spinbutton', { name: 'Seats Offered' })).toBeVisible()
    });
    test('role-flip resets seats', async({page, context})=>{
        await openJoinForm(page, context)
        const roles = page.getByRole('group', { name: 'Hiker Role' })
        await roles.getByText('Driver').click()
        await page.getByRole('spinbutton', {name: 'Seats Offered'}).fill('3')
        await expect(page.getByRole('spinbutton', { name: 'Seats Offered' })).toHaveValue('3')
        await roles.getByText('Rider').click()
        await roles.getByText('Driver').click()
        await expect(page.getByRole('spinbutton', { name: 'Seats Offered' })).toHaveValue('0')


    })
})



export async function openJoinForm(page: Page, context: BrowserContext){
    const actorHost = await createActor()
    const outing = await createOuting(actorHost)
    const actorHiker = await createActor()
    await actorInBrowser(actorHiker, context)
    await page.goto(`/outings/${outing.id}`)
    await page.getByRole('button', {name:'Request to join'}).click()
    return{outing, actorHiker, actorHost}
}