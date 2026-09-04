import {test, expect} from '@playwright/test'
import {expectLoggedIn, openNav} from "../fixtures/assertions.ts";
import {createUnverifiedActor} from "../fixtures/actor.ts";
import {WEB_URL} from "../fixtures/config.ts";
import {EMAIL_NEED_VERIFICATION, RESEND_RESPONSE} from "../fixtures/copy.ts";
import {mintVerificationToken} from "../fixtures/db.ts";
import {verifyEmail} from "../fixtures/authApiHelper.ts";


test.describe("login gate", ()=>{
    test('login to unverified account validates error, resend email and verify', async ({page}) =>{
        const actor = await createUnverifiedActor()
        await page.goto("/")
        await openNav(page)
        await page.getByRole('link', {name: 'Log in'}).click()
        await expect(page).toHaveURL(`${WEB_URL}/login`)

        const {name, email, password} = actor.user
        await page.getByRole('textbox', {name:'Email'}).fill(email)
        await page.getByRole('textbox', {name:'Password'}).fill(password)
        await page.getByRole('button', {name:'Log in'}).click()
        await expect(page.getByText(EMAIL_NEED_VERIFICATION)).toBeVisible()
        await page.getByRole('button', { name: 'Resend verification' }).click()
        await expect(page.getByText(RESEND_RESPONSE)).toBeVisible()

        const raw = await mintVerificationToken(actor.hikerID)
        await verifyEmail(actor.api, raw)

        await page.getByRole('textbox', {name:'Password'}).fill(password)   // form state may persist; re-fill what's needed
        await page.getByRole('button', {name:'Log in'}).click()
        await expectLoggedIn(page, name)
    } )
})