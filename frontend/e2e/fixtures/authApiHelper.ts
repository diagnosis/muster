import type {APIRequestContext} from "@playwright/test";

export async function verifyEmail(api:APIRequestContext, raw:string){
    const res = await api.post('/api/auth/verify-email', {data: {token: raw}})
    if (!res.ok()){
        throw new Error(`email verification failed: ${res.status()} ${await res.text()}`)
    }
    return res
}

