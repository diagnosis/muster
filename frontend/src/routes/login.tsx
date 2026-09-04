// src/routes/login.tsx

import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {useState} from "react";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api";
import type {MeResponse, ResendEmailResponse} from "@/types";
import styles from "@/routes/form.module.css"
import {requireGuest} from "@/queries.ts";
import {CODE_EMAIL_NOT_VERIFIED} from "@/lib/copy.ts";

export const Route = createFileRoute('/login')({
    beforeLoad: async ({context}) => requireGuest(context.queryClient),
    component: LoginPage
})


export function LoginPage(){
    const queryClient = useQueryClient()
    const navigate = useNavigate()
    const [email, setEmail] = useState("")
    const [password, setPassword] = useState("")

    const login = useMutation({
        mutationFn: async(creds : {email: string, password: string}) => {
            const res = await apiClient.post<MeResponse>('/api/auth/login', creds)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () =>{
            queryClient.invalidateQueries({queryKey:['me']})
            navigate({to: '/'})
        }
    })

    const resendMutation = useMutation({
        mutationFn: async (email:{email:string}) => {
            const res = await apiClient.post<ResendEmailResponse>('/api/auth/resend-verification', email)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
    }
    })


    function handleSubmit(e : React.SubmitEvent){
        e.preventDefault()
        login.mutate({email, password})

    }
    return <>
        <form className={styles.form}
            onSubmit={handleSubmit}>
            <h1>Log in</h1>
            <label className={styles.label}>Email
                <input
                    type="text"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                />
            </label>
            <label className={styles.label}>Password
                <input
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                />
            </label>
            <button className="btn-primary"
                type="submit" disabled={login.isPending}>Log in</button>
        </form>
        <div className={styles.messageContainer}>
            {login.error &&
                <p className={styles.error}>{login.error.message}</p>}
            {login.error instanceof ApiRequestError && login.error.apiError.status === CODE_EMAIL_NOT_VERIFIED &&
                <button className={styles.quietBtn} disabled={resendMutation.isPending} onClick={()=>resendMutation.mutate({email})}>Resend verification</button>}
            {resendMutation.error && <p className={styles.error}>{resendMutation.error.message}</p>}
            {resendMutation.isSuccess&& <p>{resendMutation.data.message}</p>}
        </div>
    </>
}

