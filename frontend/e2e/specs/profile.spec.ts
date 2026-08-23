import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor, uniqueName} from "../fixtures/actor.ts";
import {openNav} from "../fixtures/assertions.ts";

test.describe('profile', ()=>{
    test('name update', async({page, context})=>{
       const actor = await createActor()
        await actorInBrowser(actor, context)
        await page.goto('/')
        await openNav(page)
        await page.getByRole('link', { name: actor.user.name }).click()
        await expect(page.getByText(actor.user.email)).toBeVisible()
        const newName = uniqueName()
        await page.getByRole('textbox', { name: 'Name' }).fill(newName)
        await page.getByRole('button', { name: 'Save changes' }).click()
        await openNav(page)
        await expect(page.getByRole('link', {name:newName})).toBeVisible()


    });
})