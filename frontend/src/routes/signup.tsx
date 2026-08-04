import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {useState} from "react";
import {useMutation} from "@tanstack/react-query";
import type {Experience, MeResponse, RegisterRequest} from "../types.ts";
import {apiClient} from "../lib/api.ts";
import styles from "./auth.module.css"

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

        <form className={styles.form} onSubmit={handleSubmit}>
            <h1>Sign Up</h1>
            <label className={styles.label}>
                Name:
                <input
                    className={styles.input}
                    type="text"
                    value={name}
                    onChange={e => setName(e.target.value)}
                />
            </label>
            <label className={styles.label}>
                Email:
                <input
                    className={styles.input}
                    type="email"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                />
            </label>
            <label className={styles.label}>
                Password:
                <input
                    className={styles.input}
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                />
            </label>
            <label className={styles.label}>
                Experience:
            </label>
            <div className={styles.radioRow}>
                <label className={`${styles.radioButton} ${experience === 'beginner' ? styles.radioButtonChecked: ''}`}>
                    <input
                        type="radio"
                        value='beginner'
                        checked={experience === 'beginner'}
                        onChange={e=> setExperience(e.target.value as Experience)}/>
                    Beginner
                </label>
                <label className={`${styles.radioButton} ${experience === 'intermediate'? styles.radioButtonChecked:''}`}>
                    <input
                        type="radio"
                        value='intermediate'
                        checked={experience === 'intermediate'}
                        onChange={e=> setExperience(e.target.value as Experience)}/>
                    Intermediate
                </label>
                <label className={`${styles.radioButton} ${experience === 'experienced' ? styles.radioButtonChecked: ''} `}>
                    <input
                        type="radio"
                        value='experienced'
                        checked={experience === 'experienced'}
                        onChange={e=> setExperience(e.target.value as Experience)}/>
                    Experienced
                </label>
            </div>
            {signup.error && <p className={styles.error}>{signup.error.message}</p>}
            <button className={styles.button} type="submit" disabled={signup.isPending}>Sign up</button>
        </form>
    </>
}
