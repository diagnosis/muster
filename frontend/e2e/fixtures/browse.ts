// showOnBrowse clicks "Load more" until the card for the given outing title is
// on the page. Browse is paginated, and other specs park their outings at the
// front of the list too, so a spec cannot assume its outing is on page one.
// Gives up after five clicks; the spec's own assertion then reports the miss.
import {expect, type Page} from "@playwright/test";

export async function showOnBrowse(page: Page, title: string): Promise<void> {
    await expect(page.getByRole('heading', {name: 'Outings', exact: true})).toBeVisible()

    const card = page.getByRole('heading', {name: title})
    const cards = page.locator('a[href^="/outings/"]')
    const more = page.getByRole('button', {name: 'Load more'})

    for (let i = 0; i < 5; i++) {
        if (await card.count() > 0) return
        if (!(await more.isVisible())) return
        const before = await cards.count()
        await more.click()
        await expect(cards).not.toHaveCount(before)   // wait for the next page to render
    }
}