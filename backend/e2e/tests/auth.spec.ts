import { test, expect } from '@playwright/test';
import {asUnverifiedUser, asUser, BASE} from '../fixtures'
import {unwrap, unwrapError} from "../envelope";
import {forgotPassword, getMe, login, logout, refresh, resetPassword, signup, verifyEmail} from "../api";
import {
  ApiErrorBody,
  CODE_EMAIL_NOT_VERIFIED, FORGOT_SENTENCE, ForgotPasswordResponse,
  MeResponse,
  RegisterRequest, ResetPasswordResponse,
  VerifyEmailResponse
} from "../types";
import {mintVerificationToken} from "../db";






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

  test("register(unverified) -> login -> 403 -> mint & verify -> 200 -> get_me -> truthy", async () => {
    const {ctx, user, id} = await asUnverifiedUser(BASE)

    let res = await login(ctx, {email:user.email, password:user.password})
    await unwrap(res, 403)

    const err  = await unwrapError(res)
    expect(err.message).toBe("Email needs to be verified.")
    expect(err.code).toBe(CODE_EMAIL_NOT_VERIFIED)

    const raw = await mintVerificationToken(id)
    res = await verifyEmail(ctx, raw)
    await unwrap(res, 200)

    res = await login(ctx, {email:user.email, password:user.password})
    await unwrap(res, 200)

    res = await getMe(ctx)
    const me = await unwrap<MeResponse>(res, 200)
    expect(me.verified_at).toBeTruthy()

  })

  test('mint_verification(ttlHours:-1) -> verify_email -> 409', async () => {
    const { ctx, user, id } = await asUnverifiedUser(BASE)
    const raw = await mintVerificationToken(id, {ttlHours:-1})
    await unwrap<VerifyEmailResponse>(verifyEmail(ctx, raw),409)

  })

  test('register(verified) -> mint&verify -> 404', async () => {
    const {ctx, user, id } = await asUser(BASE)
    const raw = await mintVerificationToken(id)
    await unwrap<VerifyEmailResponse>(verifyEmail(ctx, raw), 404)
  })

  test("register(unverified) -> mint_raw -> verify -> 200 -> verify ->  409", async () =>{
    const { ctx, user, id} = await asUnverifiedUser(BASE)
    const raw = await mintVerificationToken(id)
    await unwrap<VerifyEmailResponse>(verifyEmail(ctx, raw), 200)
    await unwrap<VerifyEmailResponse>(verifyEmail(ctx, raw), 409)
  })

  test('verify_email(raw:garbage)->404', async () => {
    const { ctx, user, id } = await asUser(BASE)
    await unwrap<VerifyEmailResponse>(verifyEmail(ctx, 'trump'), 404)
  })
  // forgot / reset password
  test("forgot-password->200", async ()=>{
    const {ctx, user, id} = await asUser(BASE)
    const res = await unwrap<ForgotPasswordResponse>(await forgotPassword(ctx, user.email), 200)
    expect(res.message).toBe(FORGOT_SENTENCE)

  })
  test("forgot-password->200 for unknown email", async () => {
    const {ctx} = await asUser(BASE)
    const res =  await unwrap<ForgotPasswordResponse>(await forgotPassword(ctx, "test@unknow.com"), 200)
    expect(res.message).toBe(FORGOT_SENTENCE)
  })
  test("mint_forgot_password -> reset-password -> 200-> login with new password -> 200 -> login with old -> 401", async()=>{
    const {ctx, user, id} = await asUser(BASE)
    const raw = await mintVerificationToken(id, {purpose:"forgot_password"})
    await unwrap<ResetPasswordResponse>(await resetPassword(ctx, "Secure123", raw), 200)
    await unwrap(await login(ctx, {email:user.email, password:"Secure123"}),200)
    await unwrap(await login(ctx, {email:user.email, password:user.password}), 401)
  })
  test("mint_forgot_password -> reset-password -> reset-password -> 409", async()=>{
    const {ctx, user, id} = await asUser(BASE)
    const raw = await mintVerificationToken(id, {purpose:"forgot_password"})
    await unwrap<ResetPasswordResponse>(await resetPassword(ctx, "Secure123", raw), 200)
    await unwrap<ResetPasswordResponse>(await resetPassword(ctx, "Secure456", raw), 409)
  })
  test("mint_garbage -> reset-password -> 404", async ()=> {
    const {ctx} = await asUser(BASE)
    const garbage = "iwilldoitoneday"
    await unwrap<ResetPasswordResponse>(await resetPassword(ctx, "Secure123", garbage),404)
  })
  test("mint_forgot_password -> reset-password with weak password -> 400", async ()=> {
    const {ctx, user, id} = await asUser(BASE)
    const weakPassword = "weak12"
    const raw = await mintVerificationToken(id, {purpose:"forgot_password"})
    await unwrap(await resetPassword(ctx, weakPassword, raw), 400)
  })

})
