// src/routes/login.tsx

import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {useState} from "react";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api";
import type {MeResponse} from "@/types";
import styles from "@/routes/form.module.css"
import {requireGuest} from "@/queries.ts";

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


    function handleSubmit(e : React.SubmitEvent){
        e.preventDefault()
        login.mutate({email, password})

    }
    return <>
        <form className={styles.form}
            onSubmit={handleSubmit}>
            <h1>Log In</h1>
            <label className={styles.label}>Email:
                <input
                    type="text"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                />
            </label>
            <label className={styles.label}>Password:
                <input
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                />
            </label>
            {login.error && <p className={styles.error}>{login.error.message}</p>}
            <button className="btn-primary"
                type="submit" disabled={login.isPending}>Log in</button>
        </form>
    </>
}

