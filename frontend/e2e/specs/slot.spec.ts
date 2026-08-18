// e2e/specs/slot.spec.ts
import { expect,test} from '@playwright/test'

import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, cancelOuting, createOuting, declineRequest} from "../fixtures/outingApiHelper.ts";
import {ACCEPTED, CANCELLED, DECLINED} from "../fixtures/copy.ts";

import {createAndJoinOuting} from "../fixtures/createAndJoinOuting.ts";

test.describe('slot testing', ()=> {
    test('accept join request', async ({page, context})=>{
        const {actorHost, actorHiker, outing, jr} = await createAndJoinOuting()
        await acceptRequest(actorHost, jr.id)

        await actorInBrowser(actorHiker, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByText(ACCEPTED)).toBeVisible()
        await expect(page.getByRole('button', { name: 'Withdraw' })).toBeVisible()

    })
    test('decline join request', async ({page, context})=>{
        const {actorHost, actorHiker, outing, jr} = await createAndJoinOuting()
        await declineRequest(actorHost, jr.id)

        await actorInBrowser(actorHiker, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByText(DECLINED)).toBeVisible()
        await expect(page.getByRole('button', { name: 'Withdraw' })).not.toBeVisible()
    })
    test('cancelled override', async({page, context})=>{
        const {actorHost, actorHiker, outing, jr} = await createAndJoinOuting()
        await acceptRequest(actorHost, jr.id)
        await cancelOuting(actorHost, outing.id)

        await actorInBrowser(actorHiker, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByText(CANCELLED)).toBeVisible()
        await expect(page.getByText(ACCEPTED)).not.toBeVisible()
        await expect(page.getByRole('button', { name: 'Withdraw' })).not.toBeVisible()
    });
    test('anonymous sees login door', async ({ page }) => {
        const host = await createActor()
        const outing = await createOuting(host)
        await page.goto(`/outings/${outing.id}`)          // no handoff — anonymous
        await expect(page.getByRole('link', { name: 'Request To join' })).toBeVisible()
    });
    test('host sees host controls', async ({ page, context }) => {
        const host = await createActor()
        const outing = await createOuting(host)
        await actorInBrowser(host, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByRole('button', { name: 'Cancel Outing' })).toBeVisible()
    });
    test('Round-trip', async ({page, context})=>{
        const actorHost = await createActor()
        const outing = await createOuting(actorHost, {max_size:6, host_seats:2})
        const actorHiker  = await createActor()
        await actorInBrowser(actorHiker, context)
        await page.goto(`/outings/${outing.id}`)

        await page.getByRole('button', { name: 'Request to join' }).click()
        const joinForm = page.getByRole('form',{name:'Join outing'})
        await expect(joinForm).toBeVisible()
        const roles = page.getByRole('group', { name: 'Hiker Role' })
        await roles.getByText('Driver').click()
        await joinForm.getByRole('spinbutton', { name: 'Seats Offered' }).fill('3')
        await joinForm.getByRole('spinbutton', { name: 'Guests' }).fill('1')
        await joinForm.getByRole('textbox', { name: 'Note to the host' }).fill('Halay Guzeldir.')
        await joinForm.getByRole('button', {name:'Request to join'}).click()
        await expect(page.getByText('Requested — waiting on host')).toBeVisible()
        await expect(page.getByRole('button', { name: 'Withdraw' })).toBeEnabled()
        await page.getByRole('button', { name: 'Withdraw' }).click()
        await expect(page.getByRole('button', { name: 'Request to join' })).toBeEnabled()
    });

})



