import {test, expect} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import { createOuting } from "../fixtures/outingApiHelper.ts";
import {COMMENT_REMOVED} from "../fixtures/copy.ts";



test.describe("add comment and like", ()=>{
    test(`post a comment -> verify it appears`, async ({page, context})=>{
        const host = await createActor()

        const outing = await createOuting(host, {max_size:6, host_seats:4})
        await actorInBrowser(host, context)
        await page.goto(`/outings/${outing.id}`)
        await page.getByRole('textbox', { name: 'Add to the discussion' }).fill("need more drivers")
        await page.getByRole('button', { name: 'Post' }).click()
        await expect(page.getByText('need more drivers')).toBeVisible()
        await page.getByRole('button', { name: 'Like' }).click()
        await expect(page.getByRole('button', { name: 'Like' })).toContainText('1')
        await expect(page.getByRole('button', { name: 'Like' })).toHaveAttribute('aria-pressed', 'true')
        await page.getByRole('button', { name: 'Reply' }).click()
        await page.getByRole('textbox', { name: 'Write a reply...' }).fill('never mind, issue solved')

        await page.getByRole('button', { name: 'Post reply' }).click()
        const row = page.getByRole('article').filter({ hasText: 'need more drivers' })
        await row.getByRole('button', { name: 'Delete' }).click()
        await page.getByRole('button', { name: 'Yes' }).click()
        await expect(page.getByText(COMMENT_REMOVED)).toBeVisible()
        await expect(page.getByText('never mind, issue solved')).toBeVisible()

    });
})