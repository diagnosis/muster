import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest} from "../fixtures/outingApiHelper.ts";


test.describe("chat message", ()=> {
    test("post message -> receive message", async ({browser}) => {
        const host = await createActor()
        const hiker = await createActor()
        const hostCtx = await browser.newContext()
        const hikerCtx = await browser.newContext()
        await actorInBrowser(host, hostCtx)
        await actorInBrowser(hiker, hikerCtx)

        const outing =await createOuting(host)
        const jr = await joinRequest(hiker, outing.id)
        await acceptRequest(host, jr.id)

        const hostPage = await hostCtx.newPage();
        const hikerPage = await hikerCtx.newPage();

        await hostPage.goto(`/outings/${outing.id}/conversation`)
        await hikerPage.goto(`/outings/${outing.id}/conversation`)

        await hostPage.getByRole("textbox", {name:"chat-box"}).fill("hello")
        await hostPage.getByRole("button", { name: "send" }).click();


        await expect(hikerPage.getByText("hello")).toBeVisible()

    })
})