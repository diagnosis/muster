// e2e/specs/create-outing.spec.ts

import {expect, test} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {createOuting} from "../fixtures/outingApiHelper.ts";

test.describe('create outing', ()=>{
    test('disabled-until-valid', async({page, context})=>{
        const actor = await createActor()
        await actorInBrowser(actor, context)
        await page.goto('/')
        await page.getByRole('link', {name:'Create outing'}).click()
        await expect(page.getByRole('heading', { name: 'Create Outing' })).toBeVisible()

        await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
        await page.getByRole('textbox', {name: 'Title'}).fill('Red Mountain')
        await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
        await page.getByRole('textbox', { name: 'Destination' }).fill('Snoqualmie Pass')
        await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
        await page.getByRole('textbox', { name: 'Meet Label' }).fill('South Bellevue Park & Ride')
        await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
        await page.getByRole('textbox', { name: 'Starts At' }).fill(datetimeLocal(48))
        await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
        await page.getByRole('spinbutton', { name: 'Max Size' }).fill('8')
        await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
        await page.getByRole('spinbutton', { name: 'Host Seats' }).fill('4')
        await expect(page.getByRole('button', {name:'Create Outing'})).toBeDisabled()
        await page.getByRole('spinbutton', { name: 'Cost per Seat' }).fill('20')
        await page.getByRole('group', { name: 'Difficulty' }).getByText('Moderate').click()
        await page.getByText('Relaxed').click()
        await expect(page.getByRole('button', {name:'Create Outing'})).toBeEnabled()
        await page.getByRole('textbox', { name: 'Notes' }).fill('yesterday is history, tomorrow is mystery and, today is gift!')
        await page.getByRole('button', {name: 'Create Outing'}).click()
        await expect(page.getByRole('button', { name: 'Cancel Outing' })).toBeEnabled()

    });
    test('seeded outing renders on browse', async ({page})=>{
        const actor = await createActor()
        const outing = (await createOuting(actor))
        await page.goto('/')
        await expect(page.getByRole('link', {name: outing.title})).toBeVisible()
    })



})

function datetimeLocal(hoursFromNow: number): string {
    const d = new Date(Date.now() + hoursFromNow * 3_600_000)
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset())  // shift so toISOString prints local wall-clock
    return d.toISOString().slice(0, 16)                    // "2026-08-16T21:30"
}