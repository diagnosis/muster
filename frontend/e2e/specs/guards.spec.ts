
// e2e/specs/guards.spec.ts
import {test, expect} from "@playwright/test";

import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {WEB_URL} from "../fixtures/config.ts";

test.describe("validate guards work as expected", ()=>{
    test('anonymous should land on login when navigates to me/outings', async({page})=> {
        await page.goto('/me/outings')
        await expect(page).toHaveURL(`${WEB_URL}/login`)
    })
    test('logged in user should land on home when attempt to go to login', async ({page, context})=>{
        const actor = await createActor()
        await actorInBrowser(actor, context)
        await page.goto('/login')
        await expect(page).toHaveURL(`${WEB_URL}/`)
        await actor.dispose()

    })
})