import {test, expect} from '@playwright/test'
import {WEB_URL} from "../fixtures/config.ts";
import {actorInBrowser, createActor, createUnverifiedActor, uniqueIdentity} from "../fixtures/actor.ts";
import {expectLoggedIn, expectLoggedOut, expectNavClosed, openNav} from "../fixtures/assertions.ts";
import {
    EMAIL_EXISTS,
    INVALID_LOGIN,
    INVALID_PASSWORD,
    SHORT_PASSWORD,
    VALIDATION_FAILED,
} from "../fixtures/copy.ts";
import {mintLoginCode} from "../fixtures/db.ts";
import {createOuting} from "../fixtures/outingApiHelper.ts";




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
        await page.getByRole('textbox', {name:'Password'}).fill("WrongPassword123!")
        await page.getByRole('button', {name:"Log in"}).click()
        await page.pause()
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

    });
    test("login with code", async ({page}) =>{
        const hiker = await createUnverifiedActor()
        await page.goto("/login")
        await page.getByRole('button', {name:'Email me a login code instead'}).click()
        await page.getByRole('textbox', {name:'email'}).fill(hiker.user.email)
        await page.getByRole('button', {name:'Send me a code'}).click()
        await expect(page.getByRole('heading', {name:'Check your email'})).toBeVisible()
        await expect(page.getByRole('button', {name: /Send again/})).toBeDisabled()
        const code = await mintLoginCode(hiker.hikerID)
        await page.getByRole('textbox', {name:'Six-digit code'}).fill(code)
        await page.getByRole('button', {name: 'Log in'}).click()
        await expectLoggedIn(page, hiker.user.name)
    })
    test("logging out in one tab signs the other tab out too", async ({context}) => {
        const hiker = await createActor()
        await actorInBrowser(hiker, context)

        const tab1 = await context.newPage()
        await tab1.goto('/me/outings')
        await expectLoggedIn(tab1, hiker.user.name)

        const tab2 = await context.newPage()
        await tab2.goto('/')
        await openNav(tab2)
        await tab2.getByRole('button', {name: 'Log out'}).click()
        await expect(tab2).toHaveURL('/')

        // tab 1 learns on its next request
        await tab1.bringToFront()
        await tab1.reload()
        await expect(tab1).toHaveURL(/\/login/)
        await expect(tab1.getByRole('link', {name: hiker.user.name})).toHaveCount(0)
    });
    test('expired access token refreshes in place', async ({page, context}) => {
        const hiker = await createActor()
        await actorInBrowser(hiker, context)
        await page.goto('/me/outings')
        await expectLoggedIn(page, hiker.user.name)

        await context.clearCookies({name: 'access_token'})   // the name your server sets
        const refreshed = page.waitForResponse(r => r.url().endsWith('/api/auth/refresh') && r.status() === 200)
        await page.reload()
        await refreshed
        await expectLoggedIn(page, hiker.user.name)
    })
    test('a guest on a public page is never bounced to login', async ({page}) => {
        const host = await createActor()
        const o = await createOuting(host)

        await page.goto(`/outings/${o.id}`)
        await page.waitForLoadState('networkidle')
        await expect(page).toHaveURL(new RegExp(`/outings/${o.id}$`))
        await expect(page.getByRole('heading', {name: o.title})).toBeVisible()
    })
})


