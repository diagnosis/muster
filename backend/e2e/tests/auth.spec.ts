import { test, expect } from '@playwright/test';
import {asUser} from '../fixtures'
import {unwrap} from "../envelope";
import {getMe, login, logout, refresh, signup} from "../api";
import {MeResponse, RegisterRequest} from "../types";


const BASE = 'http://localhost:8088'





test.describe('auth', ()=>{

  test('me -> logout -> me 401 -> login -> me again', async() =>{
    const { ctx, user } = await asUser(BASE);

    const me = await unwrap<MeResponse>(getMe(ctx), 200)

    expect(me.email).toBe(user.email);

    let res = await logout(ctx)
    expect(res.status()).toBe(200)

    res = await getMe(ctx)
    expect(res.status()).toBe(401)

    res = await login(ctx, {email:user.email,password:user.password})
    expect(res.status()).toBe(200)

    res = await getMe(ctx)
    expect(res.status()).toBe(200);

  });
  test('duplicate email -> 409', async () => {
    const { ctx, user } = await asUser(BASE);
    const sameRegisterInput: RegisterRequest = {
       email: user.email,
      password: user.password,
      name: user.name,
      experience: user.experience,
    }
    const res = await signup(ctx, sameRegisterInput)
    expect(res.status()).toBe(409)
  })
  test('wrong password -> 401', async () => {
    const { ctx, user } = await asUser(BASE);
    const res = await login(ctx, {email:user.email, password:"WrongPass1234!"})
    expect(res.status()).toBe(401)
  })

  test('refresh -> 200 -> me -> 200 -> refresh again -> 200',  async () => {
    const {ctx, user} = await asUser(BASE)

    let res = await refresh(ctx)
    expect(res.status()).toBe(200)
    const me = await unwrap<MeResponse>(getMe(ctx), 200)
    expect(me.email).toBe(user.email);

    res = await refresh(ctx)
    expect(res.status()).toBe(200)

  })

  test('garbage access_token -> refresh -> 200', async () => {
    const {ctx, user} = await asUser(BASE)

    // tampering: explicit Cookie header overrides the jar for this request
    let res = await getMe(ctx, {Cookie:'access_token=garbage'})

    expect(res.status()).toBe(401)

    res = await refresh(ctx)
    expect(res.status()).toBe(200)

  })

  test('logout -> refresh -> 401', async () => {
    const {ctx, user} = await asUser(BASE)

    let res = await logout(ctx)
    expect(res.status()).toBe(200)

    res = await refresh(ctx)
    expect(res.status()).toBe(401)

  })

  test('garbage refresh_token -> refresh -> 401',  async ()=> {
    const {ctx,user} = await asUser(BASE)
    // tampering: explicit Cookie header overrides the jar for this request
    let res = await refresh(ctx, {Cookie:'refresh_token=wrongVal'})
    expect(res.status()).toBe(401)

  })

})
