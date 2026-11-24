
// src/routes/signup.tsx
import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {useState} from "react";
import {useMutation} from "@tanstack/react-query";
import type {Experience, MeResponse, RegisterRequest} from "@/types.ts";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import styles from "@/routes/form.module.css"
import {requireGuest} from "@/queries.ts";
import {PasswordInput} from "@/components/PasswordInput.tsx";

export const Route = createFileRoute('/signup')({
    beforeLoad: async ({context}) => requireGuest(context.queryClient),
    component: SignupPage,
})

export function SignupPage() {
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
                throw new ApiRequestError(res.error, res.httpStatus)
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
    const errDetails =signup.error instanceof ApiRequestError ? signup.error.apiError.details : undefined
    return <>

        <form className={styles.form} onSubmit={handleSubmit}>
            <h1>Sign up</h1>
            <label className={styles.label}>
                Name
                <input
                    type="text"
                    value={name}
                    onChange={e => setName(e.target.value)}
                />
            </label>
            <label className={styles.label}>
                Email
                <input
                    type="email"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                    aria-invalid={!!errDetails?.email}
                    aria-describedby={errDetails?.email ? 'signup-email-error' : undefined}
                />
            </label>
            {errDetails?.email && <p id={"signup-email-error"} className={styles.fieldError}>{errDetails.email}</p>}
            <label className={styles.label}>
                Password
                <PasswordInput
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                    aria-invalid={!!errDetails?.password}
                    aria-describedby={errDetails?.password ? 'signup-password-error': undefined}
                />
            </label>
            {errDetails?.password && <p id={"signup-password-error"} className={styles.fieldError}>{errDetails.password}</p>}
            <fieldset className={styles.fieldset}>
                <legend className={styles.legend}>Experience</legend>
                    <div className={styles.radioRow}>
                        <label className={`${styles.radioButton} ${experience === 'beginner' ? styles.radioButtonChecked: ''}`}>
                            <input
                                name={"experience"}
                                type="radio"
                                value='beginner'
                                checked={experience === 'beginner'}
                                onChange={e=> setExperience(e.target.value as Experience)}/>
                            Beginner
                        </label>
                        <label className={`${styles.radioButton} ${experience === 'intermediate'? styles.radioButtonChecked:''}`}>
                            <input
                                name={"experience"}
                                type="radio"
                                value='intermediate'
                                checked={experience === 'intermediate'}
                                onChange={e=> setExperience(e.target.value as Experience)}/>
                            Intermediate
                        </label>
                        <label className={`${styles.radioButton} ${experience === 'experienced' ? styles.radioButtonChecked: ''} `}>
                            <input
                                name={"experience"}
                                type="radio"
                                value='experienced'
                                checked={experience === 'experienced'}
                                onChange={e=> setExperience(e.target.value as Experience)}/>
                            Experienced
                        </label>
                    </div>

            </fieldset>

            {signup.error && <p role={'alert'} className={styles.error}>{signup.error.message}</p>}
            <button className="btn-primary" type="submit" disabled={signup.isPending || name==="" || password.length <=3 || email===""}>Sign up</button>
        </form>
    </>
}
