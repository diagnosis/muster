import {expect, test} from '@playwright/test'
import {createActor} from "../fixtures/actor.ts";
import {PASS_CHANGE_SUCCESS, RESET_EMAIL_TEXT} from "../fixtures/copy.ts";
import {mintVerificationToken} from "../fixtures/db.ts";
import {expectLoggedIn} from "../fixtures/assertions.ts";

test.describe("forgot password", ()=>{
    test(`verified actor -> login page -> fill email -> click on forgot password -> verify success text -> 
    mint_forgot_password -> goto reset-password page with token -> fill password and confirm password -> 
    click reset password -> validate message and button`, async ({page})=>{
        const actor = await createActor()
        await page.goto("/login")
        await page.getByRole('textbox', { name: 'Email' }).fill(actor.user.email)
        await page.getByRole('button', { name: 'Forgot password' }).click()
        await expect(page.getByText(RESET_EMAIL_TEXT)).toBeVisible()
        const raw = await mintVerificationToken(actor.hikerID, {purpose:'forgot_password'})
        await page.goto(`/reset-password?token=${raw}`)
        const newPassword = "Secure123"
        await page.getByRole('textbox', { name: 'Password', exact: true }).fill(newPassword)
        await page.getByRole('textbox', { name: 'Confirm password' }).fill(newPassword)
        await page.getByRole('button', { name: 'Reset password' }).click()
        await expect(page.getByText(PASS_CHANGE_SUCCESS)).toBeVisible()
        await expect(page.getByRole('link', { name: 'Go to login' })).toBeVisible()
        await page.getByRole('link', { name: 'Go to login' }).click()
        await page.getByRole('textbox', { name: 'Email' }).fill(actor.user.email)
        await page.getByRole('textbox', { name: 'Password' }).fill(newPassword)
        await page.getByRole('button', { name: 'Log in' }).click()
        await expectLoggedIn(page, actor.user.name)
    })
})