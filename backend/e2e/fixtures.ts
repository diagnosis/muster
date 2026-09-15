// e2e/fixtures.ts
import { request, APIRequestContext } from '@playwright/test'
import {Hiker, RegisterRequest} from "./types";
import {unwrap} from "./envelope";
import {mintVerificationToken} from "./db";
import {verifyEmail} from "./api";


export const BASE = process.env.BASE_URL ?? 'http://localhost:8088';

export async function asUnverifiedUser(
    baseURL: string,
    overrides: Partial<RegisterRequest> = {},): Promise<{ ctx: APIRequestContext; user: RegisterRequest, id: string }>{
    const {name, email} = uniqueIdentity()
    const user: RegisterRequest = {
        email: email,
        password: 'Passw0rd!123',
        name: name,
        experience: 'beginner',
        ...overrides,
    };

    const ctx = await request.newContext({ baseURL });
    let res = await ctx.post('/api/auth/signup', { data: user })
    const hiker = await unwrap<Hiker>(res, 201)

    return { ctx, user, id: hiker.id };
}
export async function asUser(baseURL: string, overrides = {}) {
    const actor = await asUnverifiedUser(baseURL, overrides)
    const raw = await mintVerificationToken(actor.id)
    await unwrap(await verifyEmail(actor.ctx, raw), 200)
    const res = await actor.ctx.post('/api/auth/login', { data: { email: actor.user.email, password: actor.user.password } })
    if (res.status() !== 200) throw new Error(`fixture login failed: ${res.status()} ${await res.text()}`)
    return actor
}

const ADJ = ['brisk', 'sunny', 'quiet', 'amber', 'swift', 'rustic', 'clever', 'stormy']
const NOUN = ['otter', 'falcon', 'cedar', 'harbor', 'lark', 'meadow', 'pebble', 'summit']



const cap = (s: string) => s[0].toUpperCase() + s.slice(1)
const pick = <T,>(a: T[]) => a[Math.floor(Math.random() * a.length)]

const HOST = 'test.dev'



export function uniqueIdentity(): { name: string; email: string } {
    const adj = pick(ADJ)
    const noun = pick(NOUN)
    const t = tag()
    return {
        name: `${cap(adj)} ${cap(noun)} ${t}`,
        email: `${adj}.${noun}.${t}@${HOST}`,
    }
}

export function uniqueName(): string {
    return uniqueIdentity().name
}

const W = Number(process.env.TEST_WORKER_INDEX ?? 0).toString(36)
let seq = 0
export function tag(): string {
    return W + (seq++ % 1296).toString(36).padStart(2, '0') + Math.random().toString(36).slice(2, 5)
}

