import {expect, test} from '@playwright/test'
import {createActor} from "../fixtures/actor.ts";
import {PASS_CHANGE_SUCCESS, RESET_EMAIL_TEXT, RESET_LINK_INVALID} from "../fixtures/copy.ts";
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
    test(`goto reset-password with a garbage token -> invalid-link message -> no password form -> back to login`, async ({page}) => {
        await page.goto('/reset-password?token=garbage-token')

        await expect(page.getByRole('alert')).toHaveText(RESET_LINK_INVALID)
        // the form must not be offered for a dead link
        await expect(page.getByRole('textbox', {name: 'Password', exact: true})).toHaveCount(0)
        await expect(page.getByRole('button', {name: 'Reset password'})).toHaveCount(0)

        await page.getByRole('link', {name: 'Back to login'}).click()
        await expect(page).toHaveURL(/\/login/)
    })

    test(`goto reset-password with no token -> invalid-link message -> no password form`, async ({page}) => {
        await page.goto('/reset-password')

        await expect(page.getByRole('alert')).toHaveText(RESET_LINK_INVALID)
        await expect(page.getByRole('textbox', {name: 'Password', exact: true})).toHaveCount(0)
    })

    test(`verified actor -> mint_forgot_password -> reset password -> open the same link again -> invalid-link message`, async ({page}) => {
        const actor = await createActor()
        const raw = await mintVerificationToken(actor.hikerID, {purpose: 'forgot_password'})

        await page.goto(`/reset-password?token=${raw}`)
        await page.getByRole('textbox', {name: 'Password', exact: true}).fill('Secure123')
        await page.getByRole('textbox', {name: 'Confirm password'}).fill('Secure123')
        await page.getByRole('button', {name: 'Reset password'}).click()
        await expect(page.getByText(PASS_CHANGE_SUCCESS)).toBeVisible()

        // a reset link works once: opening it again shows the dead-link page, not the form
        await page.goto(`/reset-password?token=${raw}`)
        await expect(page.getByRole('alert')).toHaveText(RESET_LINK_INVALID)
        await expect(page.getByRole('textbox', {name: 'Password', exact: true})).toHaveCount(0)
    })

})