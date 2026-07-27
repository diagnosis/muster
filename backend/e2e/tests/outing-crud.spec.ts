import {expect, test} from '@playwright/test'
import {asUser} from "../fixtures";
import {cancelOuting, createOuting, updateOuting} from "../api";
import {OutingResponse} from "../types";

const BASE = "http://localhost:8088"

test.describe("outing-crud actions", ()=> {
    test("create validation rejects bad inputs", async () =>{
        const { ctx: ctxHost} = await asUser(BASE)
        let res = await createOuting(ctxHost, {starts_at: new Date(Date.now()+60*60*1000).toISOString()})
        expect(res.status()).toBe(400)
        let errMessage = (await res.json())['error'].message
        expect(errMessage).toBe("outing has to be at least 24 hours in advance")

        res = await createOuting(ctxHost, {max_size:1})
        expect(res.status()).toBe(400)
        errMessage = (await res.json())['error'].message
        expect(errMessage).toBe("outing size has to be at least 2")

        res = await createOuting(ctxHost, {difficulty:"extreme"})
        expect(res.status()).toBe(400)
        errMessage = (await res.json())['error'].message
        expect(errMessage).toBe("invalid difficulty input")
    });
    test("patch merges fields, untouched fields survive", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        let res =  await createOuting(ctxHost)
        expect(res.status()).toBe(201)
        const outingData: OutingResponse = (await res.json())['data']

        res = await updateOuting(ctxHost, outingData.id, {title:'Mt. Townsend', max_size:4, destination:outingData.destination})
        expect(res.status()).toBe(200)
        const updatedRes: OutingResponse = ( await res.json())['data']
        expect(updatedRes.title).toBe('Mt. Townsend')
        expect(updatedRes.max_size).toBe(4)
        expect(updatedRes.destination).toBe(outingData.destination)
        expect(updatedRes.id).toBe(outingData.id)
        expect(updatedRes.host_seats).toBe(outingData.host_seats)
    });
    test("patch guards - nonhost patches", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        const {ctx: ctxOutsider} = await asUser(BASE)

        let res =  await createOuting(ctxHost)
        expect(res.status()).toBe(201)
        const outingData: OutingResponse = (await res.json())['data']

        res = await updateOuting(ctxOutsider, outingData.id, {title:'recep in the house'})
        expect(res.status()).toBe(403)
    })
    test("patch guards - host patches after cancel", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        let res =  await createOuting(ctxHost)
        expect(res.status()).toBe(201)
        const outingData: OutingResponse = (await res.json())['data']

        res = await cancelOuting(ctxHost, outingData.id)
        expect(res.status()).toBe(200)

        res = await updateOuting(ctxHost, outingData.id, {title:'Mt. Townsend', max_size:4})
        expect(res.status()).toBe(409)

    })

    test("patches guards - host patches starts at +1h", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        let res =  await createOuting(ctxHost)
        expect(res.status()).toBe(201)
        const outingData: OutingResponse = (await res.json())['data']
        res = await updateOuting(ctxHost, outingData.id, {starts_at: new Date(Date.now()+60*60*1000).toISOString()})
        expect(res.status()).toBe(400)
    })

    test(" cancel already canceled outing", async () => {
        const {ctx: ctxHost} = await asUser(BASE)
        let res =  await createOuting(ctxHost)
        expect(res.status()).toBe(201)
        const outingData: OutingResponse = (await res.json())['data']

        res = await cancelOuting(ctxHost, outingData.id)
        expect(res.status()).toBe(200)

        res = await  cancelOuting(ctxHost, outingData.id)
        expect(res.status()).toBe(409)
    })
})