// src/components/ProfileForm.tsx

import type {Experience, MeInputRequest, MeResponse} from "../types.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {useState} from "react";
import styles from "../routes/form.module.css";
import {apiClient} from "../lib/api.ts";

interface ProfileFormProps{
    me: MeResponse
}
export function ProfileForm({me}: ProfileFormProps){
    const qc = useQueryClient()
    const [hikerName, setHikerName] = useState(me.name?? "")
    const [hikerExperience, setHikerExperience] = useState<Experience| undefined>(me.experience)

    const meMutation = useMutation(
        {
            mutationFn: async (body:MeInputRequest) =>{
                const res = await apiClient.patch<MeResponse>(`/api/me/profile`, body)
                if (res.ok){
                    return res.data
                }
                throw new Error(res.error.message)
            },
            onSuccess: ()=>{
                qc.invalidateQueries({queryKey: ["me"]})
            }
        }
    )

    function handleProfileUpdate(e:React.SubmitEvent){
        e.preventDefault()
        // if (!window.confirm("Update your profile?")) return
        meMutation.mutate({ name: hikerName, experience: hikerExperience })
    }
    const isDirty =
        hikerName !== (me.name ?? "") ||
        hikerExperience !== me.experience
    const isDisabled =
        !hikerName ||
        !hikerExperience ||
        !isDirty ||
        meMutation.isPending
    return <div className='profileCard'>
        <form className={styles.form} onSubmit={handleProfileUpdate}>
            <label className={styles.label}>
                Id:
                <p className={styles.input}>
                    {me.id}
                </p>
            </label>
            <label className={styles.label}>
                Email:
                <p className={styles.input}>
                    {me.email}
                </p>
            </label>
            <label className={styles.label}>
                Name:
                <input
                    className={styles.input}
                    type="text"
                    value={hikerName}
                    onChange={e=>setHikerName(e.target.value)}
                />
            </label>
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

            <button type={"submit"} className={styles.button} disabled={isDisabled}>Submit Changes</button>
            {meMutation.error&&<p className={styles.error}>{meMutation.error.message}</p>}
        </form>
    </div>

}