import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest, postMessage} from "../fixtures/outingApiHelper.ts";


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

    });
    test("post message(api will take care) -> delete message; owner deletes own, host deletes all", async ({browser})=> {
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

        const hostMessage = await postMessage(host,  outing.conversation_id, {body:"yo"})
        let memberMessage = await postMessage(hiker, outing.conversation_id, {body:"whats up"})

        await hostPage.goto(`/outings/${outing.id}/conversation`)
        await hikerPage.goto(`/outings/${outing.id}/conversation`)
        await hostPage.getByRole("button", { name: `delete message ${memberMessage.id}` }).click()
        await expect(hikerPage.getByText("whats up")).not.toBeVisible()
        memberMessage = await postMessage(hiker, outing.conversation_id, {body:"why did you delete my message?"})
        await expect(hostPage.getByText("why did you delete my message?")).toBeVisible()
        await hikerPage.getByRole("button", { name: `delete message ${memberMessage.id}` }).click()
        await expect(hostPage.getByText("why did you delete my message?")).not.toBeVisible()
        await expect(hikerPage.getByRole("button", { name: `delete message ${hostMessage.id}` })).not.toBeVisible()


    })
})