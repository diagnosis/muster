import {ApiEnvelope, ApiErrorBody} from "./types";
import {APIResponse} from '@playwright/test'

export async function unwrap<T>(
    input: Promise<APIResponse> | APIResponse,
    expectedStatus: number,
){
   const res = await input
    if (res.status() !== expectedStatus){
        throw new Error(`expected ${expectedStatus}, got ${res.status()}: ${await res.text()}`)
    }
    return ((await res.json()) as ApiEnvelope<T>).data
}

export async function unwrapError(res: APIResponse): Promise<ApiErrorBody['error']>{
    return ((await res.json()) as ApiErrorBody).error
}