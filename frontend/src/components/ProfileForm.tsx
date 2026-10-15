// src/components/ProfileForm.tsx

import type {Experience, MeInputRequest, MeResponse} from "@/types.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {useState} from "react";
import styles from "@/routes/form.module.css";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import {useForgotPasswordMutation} from "@/queries/auth.ts";

interface ProfileFormProps{
    me: MeResponse
}
export function ProfileForm({me}: ProfileFormProps){
    const qc = useQueryClient()
    const [hikerName, setHikerName] = useState(me.name?? "")
    const [hikerExperience, setHikerExperience] = useState<Experience>(me.experience)

    const meMutation = useMutation(
        {
            mutationFn: async (body:MeInputRequest) =>{
                const res = await apiClient.patch<MeResponse>(`/api/me/profile`, body)
                if (res.ok){
                    return res.data
                }
                throw new ApiRequestError(res.error, res.httpStatus)
            },
            onSuccess: ()=>{
                qc.invalidateQueries({queryKey: ["me"]})
            }
        }
    )
    const forgotPasswordMutation = useForgotPasswordMutation()
    function handleProfileUpdate(e:React.SubmitEvent){
        e.preventDefault()
        meMutation.mutate({ name: hikerName, experience: hikerExperience })
    }
    const isDirty =
        hikerName !== (me.name ?? "") ||
        hikerExperience !== me.experience
    const isDisabled =
        !hikerName ||
        !isDirty ||
        meMutation.isPending

    return <div>

        <form className={styles.form} onSubmit={handleProfileUpdate}>
            <h1>Profile</h1>
            <div className={styles.readOnlyRow}>
                <span className={styles.readOnlyLabel}>Email</span>
                <span>{me.email}</span>
            </div>
            <label className={styles.label}>
                Name
                <input
                    type="text"
                    value={hikerName}
                    onChange={e=>setHikerName(e.target.value)}
                />
            </label>
            <fieldset className={styles.fieldset}>
                <legend className={styles.legend}>Experience</legend>
                <div className={styles.radioRow}>
                    <label className={`${styles.radioButton} ${hikerExperience==='beginner'? styles.radioButtonChecked:""}`}>
                        <input
                            name="experience"
                            type="radio"
                            value="beginner"
                            checked={hikerExperience==='beginner'}
                            onChange={e => setHikerExperience(e.target.value as Experience)}
                        />
                        Beginner
                    </label>
                    <label className={`${styles.radioButton} ${hikerExperience==='intermediate'? styles.radioButtonChecked:""}`}>
                        <input
                            name="experience"
                            type="radio"
                            value="intermediate"
                            checked={hikerExperience==='intermediate'}
                            onChange={e => setHikerExperience(e.target.value as Experience)}
                        />
                        Intermediate
                    </label>
                    <label className={`${styles.radioButton} ${hikerExperience==='experienced'? styles.radioButtonChecked:""}`}>
                        <input
                            name="experience"
                            type="radio"
                            value="experienced"
                            checked={hikerExperience==='experienced'}
                            onChange={e => setHikerExperience(e.target.value as Experience)}
                        />
                        Experienced
                    </label>
                </div>
            </fieldset>
            {meMutation.error&&<p className={styles.error}>{meMutation.error.message}</p>}
            <button type={"submit"} className="btn-primary" disabled={isDisabled}>Save changes</button>
            {!isDirty && <p className={styles.hint}>Nothing to save yet.</p>}
        </form>
        <section className={`${styles.form} top-margin`} aria-labelledby="password-heading">
            <h2 id="password-heading" className={styles.sectionTitle}>Password</h2>
            <p className={styles.hint}>
                We'll email you a link to set a new one. Using it signs you out everywhere, including here.
            </p>
            {forgotPasswordMutation.isSuccess
                ? <p className={styles.readOnlyRow}>Link sent to {me.email}. Check your inbox, and spam if it isn't there.</p>
                : <button type="button" className={styles.quietAction}
                          disabled={forgotPasswordMutation.isPending}
                          onClick={() => forgotPasswordMutation.mutate({email: me.email})}>
                    Send reset link
                </button>}
            {forgotPasswordMutation.isError && <p className={styles.error}>{forgotPasswordMutation.error.message}</p>}
        </section>
    </div>

}