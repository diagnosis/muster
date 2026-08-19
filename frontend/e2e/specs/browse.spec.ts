import {test, expect} from '@playwright/test'
import {createActor} from "../fixtures/actor.ts";
import {createOuting} from "../fixtures/outingApiHelper.ts";

test('anonymous visitor sees browse list', async ({ page }) => {
    const host = await createActor()
    const outing = await createOuting(host, { cost_per_seat_cents: 900 })                                  // has cost
    const freeOuting = await createOuting(host, { cost_per_seat_cents: 0 })      // zero-cost
    await page.goto('/')
    const paidCard = page.getByRole('link', { name: outing.title })
    const freeCard = page.getByRole('link', { name: freeOuting.title })
    await expect(paidCard).toBeVisible()
    await expect(freeCard).toBeVisible()
    await expect(freeCard).not.toContainText('$')          // cost renders only when nonzero
    await expect(paidCard).toContainText('/seat')          // and does render when nonzero
})