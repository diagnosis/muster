// e2e/api.ts
import { APIRequestContext } from '@playwright/test';
import {UpdateInput, UpdateMeInput} from "./types";
import {RegisterRequest} from "./types";

export const defaultOuting = () => ({
    title: 'Red Mountain',
    destination: 'Snoqualmie Pass',
    meet_label: 'Bellevue P&R',
    starts_at: new Date(Date.now() + 48 * 60 * 60 * 1000).toISOString(),
    max_size: 6,
    host_seats: 2,
    cost_per_seat_cents: 0,
    difficulty: 'moderate',
    pace: 'relaxed',
});

export const createOuting = (ctx: APIRequestContext, overrides = {}) =>
    ctx.post('/api/outings', { data: { ...defaultOuting(), ...overrides } });

export const requestJoin = (ctx: APIRequestContext, outingID: string, overrides = {}) =>
    ctx.post(`/api/outings/${outingID}/requests`, {
        data: { role: 'rider', seats_offered: 0, guests: 0, ...overrides },
    });

export const accept = (ctx: APIRequestContext, requestID: string) =>
    ctx.post(`/api/requests/${requestID}/accept`);

export const withdraw  = (ctx: APIRequestContext, outingID: string)=>
    ctx.delete(`/api/outings/${outingID}/requests/me`)

export const decline = (ctx: APIRequestContext, requestID: string) =>
    ctx.post(`/api/requests/${requestID}/decline`)

export const removeMember= (ctx: APIRequestContext, requestID: string) =>
    ctx.delete(`/api/requests/${requestID}/member`)

export const getDetail = (ctx: APIRequestContext, outingID: string)=>
    ctx.get(`/api/outings/${outingID}`)

export const updateOuting = (ctx: APIRequestContext, outingID: string, data:UpdateInput)=>
    ctx.patch(`/api/outings/${outingID}`, {data})

export const cancelOuting = (ctx: APIRequestContext, outingID: string) =>
    ctx.post(`/api/outings/${outingID}/cancel`)

export const getMe = (ctx: APIRequestContext, headers?: Record<string, string>) =>
    ctx.get('/api/auth/me', {headers})
export const logout = (ctx:APIRequestContext) =>
    ctx.post('/api/auth/logout')
export const login = (ctx:APIRequestContext, input :{email:string, password:string}) =>
    ctx.post('/api/auth/login', {data: input})
export const refresh = (ctx:APIRequestContext, headers?: Record<string,string>) =>
    ctx.post('/api/auth/refresh', {headers})
export const signup = (ctx:APIRequestContext, input:RegisterRequest)=>
    ctx.post('/api/auth/signup', {data:input})


export const listOutingRequests = (ctx:APIRequestContext, outingID:string) =>
    ctx.get(`/api/outings/${outingID}/requests`)

export const updateProfile = (ctx: APIRequestContext, data:UpdateMeInput) =>
    ctx.patch('/api/me/profile', {data})

export const getHiker = (ctx: APIRequestContext, hikerID:string)=>
    ctx.get(`/api/hikers/${hikerID}`)

export const verifyEmail = (ctx:APIRequestContext, token:string) =>
    ctx.post("/api/auth/verify-email", {data: {token}})