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
        await page.getByRole('link', { name: 'Create outing' }).click()
        await fillCreateForm(page, outing)
    });
    test('create lands on fresh detail', async ({page, context})=>{
        const actor = await createActor()
        await actorInBrowser(actor, context)
        await page.goto('/')
        await openNav(page)
        const outing = uniqueOuting()
        await page.getByRole('link', { name: 'Create outing' }).click()
        await fillCreateForm(page, outing)                      // ← the shared fill
        await page.getByRole('button', { name: 'Create outing' }).click()
        await expect(page).toHaveURL(/\/outings\/[0-9a-f-]+/)
        await expect(page.getByRole('heading', { name: outing.title })).toBeVisible()

    })
    test('seeded outing renders on browse', async ({page})=>{
        const actor = await createActor()
        const outing = (await createOuting(actor))
        await page.goto('/')
        await expect(page.getByRole('link', {name: outing.title})).toBeVisible()
    })



})

