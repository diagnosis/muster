import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {useState} from "react";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {Experience, MeResponse, RegisterRequest} from "../types.ts";
import {apiClient} from "../lib/api.ts";

export const Route = createFileRoute('/signup')({
  component: SignupPage,
})

function SignupPage() {
    const navigate = useNavigate()
    const [email, setEmail] = useState("")
    const [password, setPassword] = useState("")
    const [name, setName] = useState("")
    const [experience, setExperience] = useState<Experience>('beginner')



    const signup = useMutation(
        {
            mutationFn: async (input:RegisterRequest)=> {
                const res = await apiClient.post<MeResponse>('/api/auth/signup', input)
                if (res.ok){
                    return res.data
                }
                throw new Error(res.error.message)
            },
            onSuccess: () => {
                navigate({to: "/login"})
            }
        }
    )

    function handleSubmit(e: React.SubmitEvent){
        e.preventDefault()
        signup.mutate({email, password, name, experience})
    }

    return <>
        <form onSubmit={handleSubmit}>
            <label>
                Name:
                <input
                    type="text"
                    value={name}
                    onChange={e => setName(e.target.value)}
                />
            </label>
            <label>
                Email:
                <input
                    type="email"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                />
            </label>
            <label>
                Password:
                <input
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                />
            </label>
            <label>
                Experience:
            </label>
            <label>
                <input
                    type="radio"
                    value='beginner'
                    checked={experience === 'beginner'}
                    onChange={e=> setExperience(e.target.value as Experience)}/>
                Beginner
            </label>
            <label>
                <input
                    type="radio"
                    value='intermediate'
                    checked={experience === 'intermediate'}
                    onChange={e=> setExperience(e.target.value as Experience)}/>
                Intermediate
            </label>
            <label>
                <input
                    type="radio"
                    value='experienced'
                    checked={experience === 'experienced'}
                    onChange={e=> setExperience(e.target.value as Experience)}/>
                Experienced
            </label>
            {signup.error && <p>{signup.error.message}</p>}
            <button type="submit" disabled={signup.isPending}>Sign in</button>
        </form>
    </>
}
