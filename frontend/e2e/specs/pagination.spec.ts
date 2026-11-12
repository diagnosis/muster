// frontend/e2e/specs/pagination.spec.ts
import {expect, test} from '@playwright/test'
import {createActor} from "../fixtures/actor.ts";
import {createOuting} from "../fixtures/outingApiHelper.ts";

// Browse pagination (backend/docs/outing/pagination.md, P5 and P9): the page
// shows one page of 20 cards and a Load more button that adds the next page.
// The spec creates 21 outings so a second page exists on an empty database.
test('browse shows one page of outings, and Load more adds the next', async ({page}) => {
    const host = await createActor()
    for (let i = 0; i < 21; i++) {
        await createOuting(host)
    }

    await page.goto('/')

    // an outing card is a link to an outing with the title as its heading;
    // the "Create outing" nav link has no heading and is not counted
    const cards = page.locator('a[href^="/outings/"]').filter({has: page.getByRole('heading')})
    await expect(cards).toHaveCount(20)

    await page.getByRole('button', {name: 'Load more'}).click()

    await expect(cards).not.toHaveCount(20)
    expect(await cards.count()).toBeGreaterThan(20)
})