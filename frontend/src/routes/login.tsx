// src/routes/login.tsx

import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {useEffect, useState} from "react";
import { ApiRequestError} from "@/lib/api";
import styles from "@/routes/form.module.css"
import {requireGuest} from "@/queries.ts";
import {CODE_EMAIL_NOT_VERIFIED} from "@/lib/copy.ts";
import {
    useForgotPasswordMutation,
    useLoginWithPasswordMutation,
    useLoginWithCodeMutation,
    useResendMutation, useRequestLoginCodeMutation
} from "@/queries/auth.ts";
import {PasswordInput} from "@/components/PasswordInput.tsx";

export const Route = createFileRoute('/login')({
    beforeLoad: async ({context}) => requireGuest(context.queryClient),
    component: LoginPage
})


export function LoginPage(){
    const navigate = useNavigate()
    const [email, setEmail] = useState("")
    const [password, setPassword] = useState("")
    const [code, setCode] = useState("")
    const [mode, setMode] = useState<'password'|'request'|'verify'>('password')
    const loginWithPasswordMutation = useLoginWithPasswordMutation()
    const loginWithCodeMutation = useLoginWithCodeMutation()
    const resendMutation = useResendMutation()
    const requestLoginCodeMutation = useRequestLoginCodeMutation()
    const forgotPasswordMutation = useForgotPasswordMutation()
    const RESEND_WINDOW_S = 30   // mirrors authtoken.resendWindow
    const [resendIn, setResendIn] = useState(0)
    useEffect(() => {
        if (resendIn <= 0) return
        const id = setInterval(() => setResendIn(s => s - 1), 1000)
        return () => clearInterval(id)
    }, [resendIn])

    function sendCode(onSent?: () => void) {
        requestLoginCodeMutation.mutate({email}, {
            onSuccess: () => { setResendIn(RESEND_WINDOW_S); onSent?.() },
        })
    }


    function handleSubmit(e : React.SubmitEvent){
        e.preventDefault()
        switch (mode){
            case "password":{
                return loginWithPasswordMutation.mutate({email, password}, {onSuccess: ()=> navigate({to:'/'})});
            }
            case "request":{
                return sendCode(() => setMode('verify'))
            }
            case "verify": {
                return loginWithCodeMutation.mutate({email, code}, {onSuccess: ()=> navigate({to:'/'})})
            }

        }
    }
    const byMode = {password: loginWithPasswordMutation, request: requestLoginCodeMutation, verify: loginWithCodeMutation}
    const active = byMode[mode]
    return <>
        <form className={styles.form} onSubmit={handleSubmit}>
            <h1>{mode === 'verify' ? 'Check your email' : 'Log in'}</h1>

            {mode === 'verify' && requestLoginCodeMutation.data &&
                <p role="status" className={styles.readOnlyRow}>
                    {requestLoginCodeMutation.data.message} It expires in 10 minutes.
                </p>}

            <label className={styles.label}>Email
                <input
                    type="email"
                    value={email}
                    readOnly={mode === 'verify'}
                    onChange={e => setEmail(e.target.value)}
                />
            </label>

            {mode === 'password' &&
                <label className={styles.label}>Password
                    <PasswordInput value={password} onChange={e => setPassword(e.target.value)}/>
                </label>}

            {mode === 'verify' &&
                <label className={styles.label}>Six-digit code
                    <input
                        className={styles.codeInput}
                        inputMode="numeric"
                        autoComplete="one-time-code"
                        maxLength={6}
                        value={code}
                        onChange={e => setCode(e.target.value)}
                    />
                </label>}

            <div className={styles.actions}>
                <button className="btn-primary" type="submit" disabled={active.isPending}>
                    {mode === 'request' ? 'Send me a code' : 'Log in'}
                </button>
                {mode === 'verify' &&
                    <button type="button" className={styles.quietBtn}
                            disabled={requestLoginCodeMutation.isPending || resendIn > 0}
                            onClick={() => sendCode()}>
                        {resendIn > 0 ? `Send again (${resendIn} s)` : 'Send a new code'}
                    </button>}
            </div>
        </form>

        <div className={styles.messageContainer}>
            {active.error &&
                <p className={styles.error}>{active.error.message}</p>}
            {active.error instanceof ApiRequestError && active.error.apiError.code === CODE_EMAIL_NOT_VERIFIED &&
                <button className={styles.quietBtn} disabled={resendMutation.isPending}
                        onClick={() => resendMutation.mutate({email})}>Resend verification</button>}
            {resendMutation.error && <p className={styles.error}>{resendMutation.error.message}</p>}
            {resendMutation.isSuccess && <p>{resendMutation.data.message}</p>}

            <div className={styles.doors}>
                <button type="button" className={styles.quietBtn}
                        onClick={() => setMode(mode === 'password' ? 'request' : 'password')}>
                    {mode === 'password' ? 'Email me a login code instead' : 'Use my password instead'}
                </button>
                {mode === 'password' &&
                    <button type="button" className={styles.quietBtn}
                            disabled={forgotPasswordMutation.isPending || email.trim() === ""}
                            onClick={() => forgotPasswordMutation.mutate({email})}>Forgot password</button>}
            </div>
            {forgotPasswordMutation.error && <p className={styles.error}>{forgotPasswordMutation.error.message}</p>}
            {forgotPasswordMutation.isSuccess && <p className={styles.readOnlyRow}>{forgotPasswordMutation.data.message}</p>}
        </div>
    </>
}

