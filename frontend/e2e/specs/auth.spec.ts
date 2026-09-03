import {test, expect} from '@playwright/test'
import {WEB_URL} from "../fixtures/config.ts";
import {actorInBrowser, createActor, uniqueIdentity} from "../fixtures/actor.ts";
import {expectLoggedIn, expectLoggedOut, expectNavClosed, openNav} from "../fixtures/assertions.ts";
import {
    EMAIL_EXISTS,
    INVALID_LOGIN,
    INVALID_PASSWORD,
    SHORT_PASSWORD,
    VALIDATION_FAILED,
} from "../fixtures/copy.ts";




test.describe("auth flow", ()=> {
    test('successful register', async ({page})=>{
        await page.goto("/")
        await openNav(page)
        await page.getByRole('link', { name: 'Sign up' }).click()

        await expect(page).toHaveURL(`${WEB_URL}/signup`)
        await expect(page.getByRole('heading', { name: 'Sign up' })).toBeVisible()

        const {name, email} = uniqueIdentity()
        await page.getByRole('textbox', { name: 'Name' }).fill(name)
        await page.getByRole('textbox', {name: 'Email'}).fill(email)
        await page.getByRole('textbox', {name:'Password'}).fill("Secure123")
        await page.getByText('Intermediate').click()
        await expect(page.getByRole('radio', { name: 'Intermediate' })).toBeChecked()
        await page.getByRole('button', {name:'Sign up'}).click()
        await expect(page).toHaveURL(`${WEB_URL}/login`)
    });

    test('email already taken - returns valid error message', async ({page})=>{
        const actor = await createActor()
        const {email} = actor.user

        await page.goto("/")
        await openNav(page)
        await page.getByRole('link', { name: 'Sign up' }).click()

        await expect(page).toHaveURL(`${WEB_URL}/signup`)
        await expect(page.getByRole('heading', { name: 'Sign up' })).toBeVisible()

        await page.getByRole('textbox', { name: 'Name' }).fill("test-error")
        await page.getByRole('textbox', {name: 'Email'}).fill(email)
        await page.getByRole('textbox', {name:'Password'}).fill("Secure123")
        await page.getByText('Intermediate').click()
        await expect(page.getByRole('radio', { name: 'Intermediate' })).toBeChecked()
        await page.getByRole('button', {name:'Sign up'}).click()
        await expect(page.getByText(EMAIL_EXISTS)).toBeVisible()

    });
    test('invalid password for registration field', async ({page})=>{
        await page.goto('/')
        await openNav(page)
        await page.getByRole("link", {name: 'Sign up'}).click()
        await expect(page).toHaveURL(`${WEB_URL}/signup`)
        await expect(page.getByRole('heading', { name: 'Sign up' })).toBeVisible()
        await page.getByRole('textbox', { name: 'Name' }).fill("test-error")
        await page.getByRole('textbox', {name:'Email'}).fill("valid@test.com")
        await page.getByText('Experienced').click()
        await expect(page.getByRole('radio', {name:'Experienced'})).toBeChecked()
        const badPass:Record<string, string> = {
            'short': SHORT_PASSWORD,
            'nouppercase1': INVALID_PASSWORD,
        }
       for (const k in badPass) {
           await page.getByRole('textbox', {name:'Password'}).fill(k)
           await page.getByRole('button',{name:'Sign up'} ).click()
           await expect(page.getByText(badPass[k])).toBeVisible()
           await expect(page.getByText(VALIDATION_FAILED)).toBeVisible()
       }

    })

    test('successful login', async ({page})=>{
        const actor = await createActor()
        await page.goto("/")
        await openNav(page)
        await page.getByRole('link', {name: 'Log in'}).click()
        await expect(page).toHaveURL(`${WEB_URL}/login`)

        const {name,email, password} = actor.user
        await page.getByRole('textbox', {name:'Email'}).fill(email)
        await page.getByRole('textbox', {name:'Password'}).fill(password)
        await page.getByRole('button', {name:'Log in'}).click()
        await expectLoggedIn(page, name)


    });
    test('failed login', async ({page})=>{
        await page.goto('/')
        await openNav(page)
        await page.getByRole('link', {name: 'Log in'}).click()
        await expect(page).toHaveURL(`${WEB_URL}/login`)

        await page.getByRole('textbox', {name:'Email'}).fill("test@test.com")
        await page.getByRole('textbox', {name:'Password'}).fill("Password123")
        await page.getByRole('button', {name:"Log in"}).click()
        await expect(page.getByText(INVALID_LOGIN)).toBeVisible()

        await page.getByRole('textbox', {name:'Email'}).fill("wrong@test.com")
        await page.getByRole('textbox', {name:'Password'}).fill("Secure123")
        await page.getByRole('button', {name:"Log in"}).click()
        await expect(page.getByText(INVALID_LOGIN)).toBeVisible()
    })
    test('successful logout', async({page, context})=> {
        const actor = await createActor()
        const {name} = actor.user
        await actorInBrowser(actor, context)
        await page.goto('/')
        await openNav(page)
        await expectLoggedIn(page, name)
        await page.getByRole('button', { name: 'Log out' }).click()
        await expectNavClosed(page)
        await expectLoggedOut(page)
    });
    test('successful refresh', async ({page, context})=>{
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
    test('failed refresh', async  ({page, context})=>{
        const actor = await createActor()
        const {name} = actor.user
        await actorInBrowser(actor, context)
        await page.goto('/')
        await expectLoggedIn(page, name)
        // mimicking bad cookie values
        await context.clearCookies()
        await context.addCookies([{ name: 'refresh_token', value: 'garbage', domain: 'localhost', path: '/' }])
        await page.reload()
        await expectLoggedOut(page)

    })
})


