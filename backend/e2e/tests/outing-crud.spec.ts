import {expect, test} from '@playwright/test'
import {asUser,BASE} from "../fixtures";
import {cancelOuting, createOuting, updateOuting} from "../api";
import {OutingResponse} from "../types";
import {unwrap, unwrapError} from "../envelope";


test.describe("outing-crud actions", ()=> {
    test("create validation rejects bad inputs", async () =>{
        const { ctx: ctxHost} = await asUser(BASE)
        let res = await createOuting(ctxHost, {starts_at: new Date(Date.now()+60*60*1000).toISOString()})
        expect(res.status()).toBe(400)
        expect((await unwrapError(res)).message).toBe('outing has to be at least 24 hours in advance');

        res = await createOuting(ctxHost, {max_size:1})
        expect(res.status()).toBe(400)
        expect((await unwrapError(res)).message).toBe("outing size has to be at least 2")

        res = await createOuting(ctxHost, {difficulty:"extreme"})
        expect(res.status()).toBe(400)
        expect((await unwrapError(res)).message).toBe("invalid difficulty input")
    });
    test("patch merges fields, untouched fields survive", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)


        const updatedRes: OutingResponse =
            await unwrap<OutingResponse>(updateOuting(ctxHost, outing.id, {title:'Mt. Townsend', max_size:4}), 200)
        expect(updatedRes.title).toBe('Mt. Townsend')
        expect(updatedRes.max_size).toBe(4)
        expect(updatedRes.id).toBe(outing.id)
        expect(updatedRes.host_seats).toBe(outing.host_seats)
        expect(updatedRes.destination).toBe(outing.destination)
    });
    test("patch guards - nonhost patches", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const {ctx: ctxOutsider} = await asUser(BASE)

        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

        const res = await updateOuting(ctxOutsider, outing.id, {title:'recep in the house'})
        expect(res.status()).toBe(403)
    })
    test("patch guards - host patches after cancel", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

        let res = await cancelOuting(ctxHost, outing.id)
        expect(res.status()).toBe(200)

        res = await updateOuting(ctxHost, outing.id, {title:'Mt. Townsend', max_size:4})
        expect(res.status()).toBe(409)

    })

    test("patches guards - host patches starts at +1h", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)
        const res = await updateOuting(ctxHost, outing.id, {starts_at: new Date(Date.now()+60*60*1000).toISOString()})
        expect(res.status()).toBe(400)
    })

    test("already canceled outing", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const outing = await unwrap<OutingResponse>(createOuting(ctxHost), 201)

        let res = await cancelOuting(ctxHost, outing.id)
        expect(res.status()).toBe(200)

        res = await  cancelOuting(ctxHost, outing.id)
        expect(res.status()).toBe(409)
    })
})