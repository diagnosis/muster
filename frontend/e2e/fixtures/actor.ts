import { request, type APIRequestContext, type BrowserContext} from '@playwright/test'
import type {RegisterRequest} from "./types.ts";
import {BASE_URL} from "./config.ts";
import {tag} from "./mint.ts";


export interface Actor{
    api: APIRequestContext
    user: RegisterRequest
    dispose: () => Promise<void>
}

export async function createActor(overrides: Partial<RegisterRequest> = {}): Promise<Actor>{
    const { name, email } = uniqueIdentity()
    const user: RegisterRequest = {
        email: email,
        password: 'Passw0rd!123',
        name: name,
        experience: 'beginner',
        ...overrides,
    }

    const api = await request.newContext({baseURL:BASE_URL})
    let res = await api.post('api/auth/signup', {data:user})
    if (!res.ok()){
        throw new Error(`login failed: ${res.status()} ${await res.text()}`)
    }
    res = await api.post('/api/auth/login', {
        data: { email: user.email, password: user.password },
    })
    if (!res.ok()) throw new Error(`login failed: ${res.status()} ${await res.text()}`)
    return {api,user, dispose: ()=> api.dispose()}
}

export async function actorInBrowser(actor: Actor, context: BrowserContext){
    const state = await actor.api.storageState()
    await context.addCookies(state.cookies)
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