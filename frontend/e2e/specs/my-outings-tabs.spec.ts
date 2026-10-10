import {expect, test, type Page} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest} from "../fixtures/outingApiHelper.ts";

const tab = (page: Page, name: RegExp) => page.getByRole('tab', {name})
const card = (page: Page, title: string) => page.getByRole('link').filter({hasText: title})

test.describe("my outings tabs", () => {
    test("host sees their outing under Upcoming, labelled as hosting", async ({page, context}) => {
        const host = await createActor()
        const outing = await createOuting(host, {max_size: 6, host_seats: 4})

        await actorInBrowser(host, context)
        await page.goto('/me/outings')

        // Upcoming is the default tab and counts the outing
        await expect(tab(page, /Upcoming/)).toHaveAttribute('aria-selected', 'true')
        await expect(tab(page, /Upcoming/)).toContainText('1')
        await expect(card(page, outing.title)).toContainText("You're hosting")

        // Past is empty: the card goes, the empty message shows
        await tab(page, /Past/).click()
        await expect(tab(page, /Past/)).toHaveAttribute('aria-selected', 'true')
        await expect(card(page, outing.title)).toHaveCount(0)
        await expect(page.getByText(/No past outings yet/)).toBeVisible()

        // and back
        await tab(page, /Upcoming/).click()
        await expect(card(page, outing.title)).toBeVisible()
    })

    test("accepted member sees the outing under Upcoming, labelled as going", async ({page, context}) => {
        const host = await createActor()
        const hiker = await createActor()
        const outing = await createOuting(host, {max_size: 6, host_seats: 4})
        const jr = await joinRequest(hiker, outing.id)
        await acceptRequest(host, jr.id)

        await actorInBrowser(hiker, context)
        await page.goto('/me/outings')

        await expect(tab(page, /Upcoming/)).toHaveAttribute('aria-selected', 'true')
        await expect(card(page, outing.title)).toContainText("You're going")
        await expect(card(page, outing.title)).not.toContainText("You're hosting")

        await tab(page, /In progress/).click()
        await expect(card(page, outing.title)).toHaveCount(0)
        await expect(page.getByText(/Nothing is happening right now/)).toBeVisible()
    })
})