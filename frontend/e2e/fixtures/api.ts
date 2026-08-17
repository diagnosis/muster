import type {APIResponse} from "@playwright/test";

export async function unwrap<T>(res: APIResponse, what: string): Promise<T> {
    if (!res.ok()) throw new Error(`${what} failed: ${res.status()} ${await res.text()}`)
    return (await res.json()).data as T
}