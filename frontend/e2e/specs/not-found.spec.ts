import {test, expect} from "@playwright/test";



test.describe("not found page validation", ()=> {
    test('validate not found web', async ({page})=>{
        await page.goto('/notfound')
        await expect(page.getByRole('heading', {name:'Dead end'})).toBeVisible()
        await expect(page.getByRole('link', {name:'Back to outings'})).toBeVisible()
    })
})