// e2e/fixtures.ts
import { request, APIRequestContext } from '@playwright/test'
import {Hiker, RegisterRequest} from "./types";
import {unwrap} from "./envelope";


export const BASE = process.env.BASE_URL ?? 'http://localhost:8088';
export async function asUser(
    baseURL: string,
    overrides: Partial<RegisterRequest> = {},
): Promise<{ ctx: APIRequestContext; user: RegisterRequest, id: string }> {
    const user: RegisterRequest = {
        email: `u${Date.now()}${Math.random().toString(36).slice(2, 6)}@test.dev`,
        password: 'Passw0rd!123',
        name: 'Test User',
        experience: 'beginner',
        ...overrides,
    };

    const ctx = await request.newContext({ baseURL });
    let res = await ctx.post('/api/auth/signup', { data: user })
    const hiker = await unwrap<Hiker>(res, 201)

    res = await ctx.post('/api/auth/login', {
        data: { email: user.email, password: user.password },
    });
    if (res.status() !== 200) throw new Error(`fixture login failed: ${res.status()} ${await res.text()}`);

    return { ctx, user, id: hiker.id };
}
