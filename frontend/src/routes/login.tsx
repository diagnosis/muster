import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {useState} from "react";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient} from "../lib/api.ts";
import type {MeResponse} from "../types.ts";

export const Route = createFileRoute('/login')({

  component: LoginPage
})


function LoginPage(){
    const queryClient = useQueryClient()
    const navigate = useNavigate()
    const [email, setEmail] = useState("")
    const [password, setPassword] = useState("")

    const login = useMutation({
        mutationFn: async(creds : {email: string, password: string}) => {
            const res = await apiClient.post<MeResponse>('api/auth/login', creds)
            if (res.ok){
                return res.data
            }
            throw new Error(res.error.message)
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
        <form onSubmit={handleSubmit}>
            <label>Email:
                <input
                    type="text"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                />
            </label>
            <label>Password:
                <input
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                />
            </label>
            {login.error && <p>{login.error.message}</p>}
            <button type="submit" disabled={login.isPending}>Log in</button>
        </form>
    </>
}

