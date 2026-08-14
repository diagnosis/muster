import {test, expect} from '@playwright/test'

test('anonymous visitor should see browse list', async ({page})=>{
    await page.goto('/')
    await expect(page.getByRole('heading', {name:'Outings'})).toBeVisible()
    await expect(page.getByRole('link', {name:/test/})).toBeVisible()
})