// e2e/specs/create-outing.spec.ts

import {expect, test} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {createOuting, uniqueOuting} from "../fixtures/outingApiHelper.ts";
import {openNav} from "../fixtures/assertions.ts";
import {fillCreateForm} from "../fixtures/mint.ts";

test.describe('create outing', ()=>{
    test('disabled-until-valid', async({page, context})=>{
        const actor = await createActor()
        await actorInBrowser(actor, context)
        await page.goto('/')
        await openNav(page)
        const outing = uniqueOuting()
        await page.getByRole('banner').getByRole('link', { name: 'Create outing' }).click()
        await fillCreateForm(page, outing, false)
        await expect(page.getByText(/ – /)).toHaveCount(0)
    });
    test('create lands on fresh detail', async ({page, context})=>{
        const actor = await createActor()
        await actorInBrowser(actor, context)
        await page.goto('/')
        await openNav(page)
        const outing = uniqueOuting()
        await page.getByRole('banner').getByRole('link', { name: 'Create outing' }).click()
        await fillCreateForm(page, outing, true)                      // ← the shared fill
        await page.getByRole('button', { name: 'Create outing' }).click()
        await expect(page).toHaveURL(/\/outings\/[0-9a-f-]+/)
        await expect(page.getByRole('heading', { name: outing.title })).toBeVisible()
        await expect(page.getByText(/\d{1,2}:\d{2} [AP]M – /)).toBeVisible()

    })
    test('seeded outing renders on browse', async ({page})=>{
        const actor = await createActor()
        const outing = (await createOuting(actor))
        await page.goto('/')
        await expect(page.getByRole('link', {name: outing.title})).toBeVisible()
    })
    test('host removes the end time on edit', async ({page, context}) => {
        const actor = await createActor()
        await actorInBrowser(actor, context)
        await page.goto('/')
        await openNav(page)
        const outing = uniqueOuting()
        await page.getByRole('banner').getByRole('link', { name: 'Create outing' }).click()
        await fillCreateForm(page, outing, true)
        await page.getByRole('button', { name: 'Create outing' }).click()
        await expect(page.getByText(/\d{1,2}:\d{2} [AP]M – /)).toBeVisible()

        await page.getByRole('link', { name: 'Edit outing' }).click()
        const box = page.getByRole('checkbox', { name: 'Set an end time' })
        await expect(box).toBeChecked()          // the form loaded the existing end
        await box.uncheck()
        await page.getByRole('button', { name: /save/i }).click()

        await expect(page.getByRole('heading', { name: outing.title })).toBeVisible()
        await expect(page.getByText(/ – /)).toHaveCount(0)
    })

})

