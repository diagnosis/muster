import { test, expect } from '@playwright/test';
import {asUser, RegisterRequest} from '../fixtures'


const BASE = 'http://localhost:8088'



interface MeResponse{
  email:string,
  name:string,
  experience:string,
  created_at: string,
  updated_at: string,
}

test.describe('auth', ()=>{

  test('me -> logout -> me 401 -> login -> me again', async() =>{
    const { ctx, user } = await asUser(BASE);

    let res = await ctx.get('/api/auth/me')
    expect(res.status()).toBe(200);
    const me: MeResponse = (await res.json())['data']
    expect(me.email).toBe(user.email);

    res = await ctx.post("/api/auth/logout")
    expect(res.status()).toBe(200)

    res = await ctx.get("/api/auth/me")
    expect(res.status()).toBe(401)

    res = await ctx.post('/api/auth/login', {
      data: {email: user.email, password:user.password}
    })
    expect(res.status()).toBe(200)

    res = await ctx.get('/api/auth/me')
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
    const res = await ctx.post('/api/auth/signup', {
      data: sameRegisterInput
    })
    expect(res.status()).toBe(409)
  })
  test('wrong password -> 401', async () => {
    const { ctx, user } = await asUser(BASE);
    const res = await ctx.post('/api/auth/login', {data:{email: user.email, password: "wrongPassword1234!"}})
    expect(res.status()).toBe(401)
  })

  test('refresh -> 200 -> me -> 200 -> refresh again -> 200',  async () => {
    const {ctx, user} = await asUser(BASE)

    let res = await ctx.post('/api/auth/refresh')
    expect(res.status()).toBe(200)

    res = await ctx.get("/api/auth/me")
    expect(res.status()).toBe(200);
    const me: MeResponse = (await res.json())['data']
    expect(me.email).toBe(user.email);

    res = await ctx.post('/api/auth/refresh')
    expect(res.status()).toBe(200)

  })

  test('garbage access_token -> refresh -> 200', async () => {
    const {ctx, user} = await asUser(BASE)

    //now garbage access_token
    let res = await ctx.get('/api/auth/me', {
      headers: {
        Cookie: 'access_token=garbage123'
      }
    })
    expect(res.status()).toBe(401)

    res = await ctx.post("/api/auth/refresh")
    expect(res.status()).toBe(200)

  })

  test('logout -> refresh -> 401', async () => {
    const {ctx, user} = await asUser(BASE)

    let res = await ctx.post('/api/auth/logout')
    expect(res.status()).toBe(200)

    res = await ctx.post('/api/auth/refresh')
    expect(res.status()).toBe(401)

  })

  test('garbage refresh_token -> refresh -> 401',  async ()=> {
    const {ctx,user} = await asUser(BASE)

    let res = await ctx.post('/api/auth/refresh', {
      headers: {
        Cookie:'refresh_token=wrongVal'
      }
    })
    expect(res.status()).toBe(401)

  })

})
