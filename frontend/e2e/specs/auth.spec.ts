import {test, expect, type Page} from '@playwright/test'
import {WEB_URL} from "../fixtures/config.ts";
import {actorInBrowser, createActor, uniqueIdentity} from "../fixtures/actor.ts";



test.describe("auth flow", ()=> {
    test('successful register', async ({page})=>{
        await page.goto("/")
        await page.getByRole('link', { name: 'Sign up' }).click()

        await expect(page).toHaveURL(`${WEB_URL}/signup`)
        await expect(page.getByRole('heading', { name: 'Sign Up' })).toBeVisible()

        const {name, email} = uniqueIdentity()
        await page.getByRole('textbox', { name: 'Name' }).fill(name)
        await page.getByRole('textbox', {name: 'Email'}).fill(email)
        await page.getByRole('textbox', {name:'Password'}).fill("Secure123")
        await page.getByText('Intermediate').click()
        await expect(page.getByRole('radio', { name: 'Intermediate' })).toBeChecked()
        await page.getByRole('button', {name:'Sign up'}).click()
        // //login page is displayed
        await expect(page).toHaveURL(`${WEB_URL}/login`)
    });

    test('email already taken - returns valid error message', async ({page})=>{
        const actor = await createActor()
        const {email} = actor.user

        await page.goto("/")
        await page.getByRole('link', { name: 'Sign up' }).click()

        await expect(page).toHaveURL(`${WEB_URL}/signup`)
        await expect(page.getByRole('heading', { name: 'Sign Up' })).toBeVisible()

        await page.getByRole('textbox', { name: 'Name' }).fill("test-error")
        await page.getByRole('textbox', {name: 'Email'}).fill(email)
        await page.getByRole('textbox', {name:'Password'}).fill("Secure123")
        await page.getByText('Intermediate').click()
        await expect(page.getByRole('radio', { name: 'Intermediate' })).toBeChecked()
        await page.getByRole('button', {name:'Sign up'}).click()
        await expect(page.getByText('email already registered')).toBeVisible()

    })

    test('user should be able to login successfully', async ({page})=>{
        const actor = await createActor()
        await page.goto("/")
        await page.getByRole('link', {name: 'Log in'}).click()
        await expect(page).toHaveURL(`${WEB_URL}/login`)

        const {name,email, password} = actor.user
        await page.getByRole('textbox', {name:'Email'}).fill(email)
        await page.getByRole('textbox', {name:'Password'}).fill(password)
        await page.getByRole('button', {name:'Log in'}).click()
        await expectLoggedIn(page, name)


    });
    test('user should be able to logout successfully', async({page, context})=> {
        const actor = await createActor()
        const {name} = actor.user
        await actorInBrowser(actor, context)
        await page.goto('/')
        await expectLoggedIn(page, name)
        await page.getByRole('button', { name: 'Log out' }).click()
        await expectLoggedOut(page)
    });
    test('user should be able to refresh refresh token to generate new access token', async ({page, context})=>{
        const actor = await  createActor()
        const {name} = actor.user
        await actorInBrowser(actor, context)
        await page.goto('/')
        await expectLoggedIn(page, name)
        // mimicking expired token
        await context.clearCookies({name: 'access_token'})

        await page.reload()
        await expectLoggedIn(page, name)
    });
    test('user should not refresh with bad refresh token', async  ({page, context})=>{
        const actor = await createActor()
        const {name} = actor.user
        await actorInBrowser(actor, context)
        await page.goto('/')
        await expectLoggedIn(page, name)
        // mimicking delete all cookies
        await context.clearCookies()
        await page.reload()
        await expectLoggedOut(page)

    })

})

async function expectLoggedIn(page:Page, name:string){
    await expect(page.getByRole('link', {name:'My outings'})).toBeVisible()
    await expect(page.getByRole('link', {name:'Create outing'})).toBeVisible()
    await expect(page.getByRole('link', {name:name})).toBeVisible()
}
async function expectLoggedOut(page:Page){
    await expect(page.getByRole('link', {name:'Log in'})).toBeVisible()
    await expect(page.getByRole('link', {name:'Sign up'})).toBeVisible()
}