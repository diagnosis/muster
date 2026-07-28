// e2e/fixtures.ts
import { request, APIRequestContext } from '@playwright/test'
import {RegisterRequest} from "./types";




export async function asUser(
    baseURL: string,
    overrides: Partial<RegisterRequest> = {},
): Promise<{ ctx: APIRequestContext; user: RegisterRequest }> {
    const user: RegisterRequest = {
        email: `u${Date.now()}${Math.random().toString(36).slice(2, 6)}@test.dev`,
        password: 'Passw0rd!123',
        name: 'Test User',
        experience: 'beginner',
        ...overrides,
    };

    const ctx = await request.newContext({ baseURL });
    let res = await ctx.post('/api/auth/signup', { data: user });
    if (res.status() !== 201) throw new Error(`signup failed: ${res.status()} ${await res.text()}`);

    res = await ctx.post('/api/auth/login', {
        data: { email: user.email, password: user.password },
    });
    if (res.status() !== 200) throw new Error(`fixture login failed: ${res.status()} ${await res.text()}`);

    return { ctx, user };
}
