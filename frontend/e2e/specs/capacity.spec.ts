import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest} from "../fixtures/outingApiHelper.ts";
import {FULL_WARNING, OUTING_FULL, SEAT_SHORTAGE} from "../fixtures/copy.ts";


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
        await expect(page.getByText(SEAT_SHORTAGE)).toBeVisible()
        await expect(page.getByText(OUTING_FULL)).not.toBeVisible()
        await expect(page.getByText(FULL_WARNING)).not.toBeVisible()

     })
})
