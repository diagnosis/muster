import {expect, test} from '@playwright/test'
import {actorInBrowser, createActor} from "../fixtures/actor.ts";
import {acceptRequest, createOuting, joinRequest, postMessage} from "../fixtures/outingApiHelper.ts";


test.describe("chat message", ()=> {
    test("post message -> receive message", async ({browser}) => {
        const host = await createActor()
        const hiker = await createActor()
        const stranger = await createActor()

        const hostCtx = await browser.newContext()
        const hikerCtx = await browser.newContext()
        const strangerCtx = await browser.newContext()

        await actorInBrowser(host, hostCtx)
        await actorInBrowser(hiker, hikerCtx)
        await actorInBrowser(stranger, strangerCtx)

        const outing =await createOuting(host)
        const jr = await joinRequest(hiker, outing.id)
        await acceptRequest(host, jr.id)

        const hostPage = await hostCtx.newPage();
        const hikerPage = await hikerCtx.newPage();
        const strangerPage = await strangerCtx.newPage()

        await hostPage.goto(`/outings/${outing.id}`)
        await hostPage.getByRole('link', {name:'Outing chat'}).click()
        await hikerPage.goto(`/outings/${outing.id}`)
        await hikerPage.getByRole('link', {name:'Outing chat'}).click()
        await strangerPage.goto(`/outings/${outing.id}`)
        await expect(strangerPage.getByRole('link', {name:"Outing chat"})).not.toBeVisible()

        await hostPage.getByRole("textbox", {name:"chat-box"}).fill("hello")
        await hostPage.getByRole("button", { name: "send" }).click();
        await expect(hikerPage.getByText("hello")).toBeVisible()

        await strangerPage.goto(`/conversations/${outing.conversation_id}`)
        await expect(strangerPage).toHaveURL(`/conversations/${outing.conversation_id}`)


    });
    test("post message using enter + shift enter for new line", async({browser})=> {
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
        await hostPage.goto(`/conversations/${outing.conversation_id}`)
        await hikerPage.goto(`/conversations/${outing.conversation_id}`)

        const hostChatBox = hostPage.getByRole("textbox", {name: "chat-box"})
        await hostChatBox.fill("hello from enter")
        await hostChatBox.press("Enter")
        await expect(hikerPage.getByText("hello from enter")).toBeVisible()

        const hikerChatBox = hikerPage.getByRole("textbox", {name: "chat-box"})
        await hikerChatBox.fill("hello maho.")
        await hikerChatBox.press("Shift+Enter")
        await hikerChatBox.pressSequentially("h r u?")
        await hikerChatBox.press("Enter")
        await expect(hostPage.getByText("hello maho.")).toBeVisible()
        await expect(hostPage.getByText("h r u?")).toBeVisible()

    })
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

        await hostPage.goto(`/outings/${outing.id}`)
        await hostPage.getByRole('link', {name:'Outing chat'}).click()
        await hikerPage.goto(`/outings/${outing.id}`)
        await hikerPage.getByRole('link', {name:'Outing chat'}).click()
        await hostPage.getByText("whats up").click()
        await hostPage.getByRole("button", { name: `delete message ${memberMessage.id}` }).click()
        await expect(hikerPage.getByText("whats up")).not.toBeVisible()
        memberMessage = await postMessage(hiker, outing.conversation_id, {body:"why did you delete my message?"})
        await expect(hostPage.getByText("why did you delete my message?")).toBeVisible()
        await hikerPage.getByText("why did you delete my message?").click()
        await hikerPage.getByRole("button", { name: `delete message ${memberMessage.id}` }).click()
        await expect(hostPage.getByText("why did you delete my message?")).not.toBeVisible()
        await expect(hikerPage.getByRole("button", { name: `delete message ${hostMessage.id}` })).not.toBeVisible()

    });
    test("host sees new join request without reload", async ({ browser }) => {
        const host = await createActor()
        const hiker = await createActor()
        const hostCtx = await browser.newContext()
        await actorInBrowser(host, hostCtx)
        const outing = await createOuting(host)

        const hostPage = await hostCtx.newPage()
        await hostPage.goto(`/outings/${outing.id}`)
        await expect(hostPage.getByText("Requests (0)")).toBeVisible()

        await joinRequest(hiker, outing.id)                       // API, no browser for the hiker

        await expect(hostPage.getByText("Requests (1)")).toBeVisible()   // no reload
        await expect(hostPage.getByRole("button", { name: "Notifications" })).toContainText("1")
    })
    test("host DMs a pending requester; bell rings; accept opens the composer", async ({ browser }) => {
        // --- seed over the API: a host, a hiker with a *pending* request (host↔pending is a DM door)
        const host = await createActor()
        const hiker = await createActor()
        const hostCtx = await browser.newContext()
        const hikerCtx = await browser.newContext()
        await actorInBrowser(host, hostCtx)
        await actorInBrowser(hiker, hikerCtx)
        const outing = await createOuting(host)
        await joinRequest(hiker, outing.id)          // left pending on purpose

        const hostPage = await hostCtx.newPage()
        const hikerPage = await hikerCtx.newPage()

        // the hiker is parked on a page that is NOT the inbox, so the bell badge
        // can only change via the notification.created poke
        await hikerPage.goto("/inbox")
        // --- host starts the DM from the requester's row (one click: POST /api/dms + navigate)
        await hostPage.goto(`/outings/${outing.id}`)
        await hostPage.getByRole("button", { name: `Message ${hiker.user.name}` }).click()
        await expect(hostPage).toHaveURL(/\/conversations\//)
        // pending from the initiator's side: one opening message allowed, then it's on them
        await expect(hostPage.getByText(/waiting/i)).toBeVisible()

        // --- the hiker's bell moves with no reload: proves KindDMRequested + the poke
        await expect(hikerPage.getByRole("button", { name: "Notifications" })).toContainText("1")

        // --- the hiker finds the request in the inbox and accepts
        await hikerPage.getByRole("link", { name:host.user.name}).click()
        await expect(hikerPage).toHaveURL(/\/conversations\//)
        await hikerPage.getByRole("button", { name: "Accept" }).click()

        // --- host's page, never reloaded, flips to a usable composer: proves dm.accepted
        await expect(hostPage.getByRole("button", {name:"Close Conversation"})).toBeVisible()
        await hostPage.getByRole("textbox", { name: "chat-box" }).fill("got microspikes?")
        await hostPage.getByRole("button", { name: "send" }).click()
        await expect(hikerPage.getByText("got microspikes?")).toBeVisible()

        await hikerPage.getByRole('textbox', {name: "chat-box"}).fill("yea, i have Kahtali Michi")
        await hikerPage.getByRole("button", { name: "send" }).click()
        await expect(hostPage.getByText("yea, i have Kahtali Michi")).toBeVisible()

    })
})