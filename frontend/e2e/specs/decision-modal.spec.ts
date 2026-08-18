// e2e/decision-modal.spec.ts

import {expect, test} from '@playwright/test'
import {createAndJoinOuting} from "../fixtures/createAndJoinOuting.ts";
import {actorInBrowser} from "../fixtures/actor.ts";
import {ACCEPT_409, CAPACITY_WARNING} from "../fixtures/copy.ts";

test.describe('host decides', ()=>{
    test('modal content renders', async({page, context})=>{
        const {actorHost,actorHiker, outing, jr} = await createAndJoinOuting({},{note:'i am poor but good hiker.'})
        await actorInBrowser(actorHost, context)
        await page.goto(`/outings/${outing.id}`)
        await page.getByRole('button', {name:actorHiker.user.name}).click()
        const modal = page.getByRole('dialog')
        await expect(modal.getByRole('heading', { name: actorHiker.user.name })).toBeVisible()
        await expect(modal.getByText(jr.role, { exact: true })).toBeVisible()
        await expect(modal.getByRole('button', { name: 'Accept' })).toBeEnabled()
        await expect(modal.getByRole('button', { name: 'Decline' })).toBeEnabled()
        await expect(modal.getByText(jr.note!)).toBeVisible()
    })
    test('decline removes row', async ({page, context})=>{
        const {actorHost,actorHiker, outing} = await createAndJoinOuting()
        await actorInBrowser(actorHost, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByRole('heading', { name: 'Requests (1)' })).toBeVisible()
        await page.getByRole('button', {name:actorHiker.user.name}).click()
        const modal = page.getByRole('dialog')
        await expect(modal.getByRole('heading', { name: actorHiker.user.name })).toBeVisible()
        await modal.getByRole('button', {name:'Decline'}).click()
        await expect(page.getByRole('heading', { name: 'Requests (0)' })).toBeVisible()
        await expect(page.getByRole('button', {name:actorHiker.user.name})).not.toBeVisible()
    });
    test('409 in modal', async({page, context})=>{
        const {actorHost, outing, actorHiker} = await createAndJoinOuting({max_size:2, host_seats:2}, {guests:1, note:"i am going to school"})
        await actorInBrowser(actorHost, context)
        await page.goto(`/outings/${outing.id}`)
        await expect(page.getByRole('heading', { name: 'Requests (1)' })).toBeVisible()
        await page.getByRole('button', {name:actorHiker.user.name}).click()
        const modal = page.getByRole('dialog')
        await modal.getByRole('button', {name: 'Accept'}).click()
        await expect(modal.getByText(CAPACITY_WARNING)).toBeVisible()
        await expect(modal.getByText(ACCEPT_409)).toBeVisible()
        await expect(modal.getByRole('button', { name: 'Accept' })).toBeEnabled()
    });


})