import {test, expect} from '@playwright/test'
import {createUnverifiedActor} from "../fixtures/actor.ts";
import {mintVerificationToken} from "../fixtures/db.ts";
import {VERIFY_EXPIRED_OR_USED, VERIFY_INVALID, VERIFY_SUCCESS} from "../fixtures/copy.ts";
import {WEB_URL} from "../fixtures/config.ts";

test.describe("verify email", ()=>{
    test('verify email success', async ({page})=> {
        const actor = await createUnverifiedActor()
        const { hikerID } = actor
        const raw = await mintVerificationToken(hikerID)
        await page.goto(`/verify-email?token=${raw}`)
        await expect(page.getByText(VERIFY_SUCCESS)).toBeVisible()
        await page.getByText(VERIFY_SUCCESS).locator('..').getByRole('link', { name: 'Log in' }).click()
        await expect(page).toHaveURL(`${WEB_URL}/login`)
    })
    test('verify email - used token', async ({page})=>{
        const actor = await createUnverifiedActor()
        const { hikerID } = actor
        const raw = await mintVerificationToken(hikerID)
        await page.goto(`/verify-email?token=${raw}`)
        await expect(page.getByText(VERIFY_SUCCESS)).toBeVisible()
        await page.goto(`/verify-email?token=${raw}`)
        await expect(page.getByText(VERIFY_EXPIRED_OR_USED)).toBeVisible()
    })
    test('verify email - empty and garbage token', async ({page}) => {
        let raw = ''
        await page.goto(`/verify-email?token=${raw}`)
        await expect(page.getByText(VERIFY_INVALID)).toBeVisible()
        raw = 'garbage-token'
        await page.goto(`/verify-email?token=${raw}`)
        await expect(page.getByText(VERIFY_INVALID)).toBeVisible()
    })
})