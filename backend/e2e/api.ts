// e2e/api.ts
import { APIRequestContext } from '@playwright/test';
import {UpdateInput} from "./types";

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

