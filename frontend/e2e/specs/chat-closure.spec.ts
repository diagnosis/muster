import {expect, test} from "@playwright/test";
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest, postMessage} from "../fixtures/outingApiHelper.ts";
import {setOutingTimes} from "../fixtures/db.ts";


test.describe("test chat closure", ()=>{
    test("happy", async ({page, context}) => {
        const host = await createActor()
        const hiker = await createActor()

        const  o = await createOuting(host)
        const jr = await joinRequest(hiker, o.id)
        await acceptRequest(host, jr.id)
        await postMessage(hiker, o.conversation_id, {body:"hi, there"})
        const day = 24 * 60 * 60 * 1000
        await setOutingTimes(o.id, new Date(Date.now() - 9 * day), new Date(Date.now() - 8 * day))

        await actorInBrowser(hiker, context)
        await page.goto(`/conversations/${o.conversation_id}`)

        await expect(page.getByText("This chat is closed")).toBeVisible()
        await expect(page.getByRole('textbox', {name: 'chat-box'})).toBeDisabled()
        await expect(page.getByRole('button', {name: 'Send'})).toBeDisabled()
        await expect(page.getByText("hi, there")).toBeVisible()
    })
})