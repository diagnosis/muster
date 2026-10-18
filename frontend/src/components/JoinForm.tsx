
// src/components/JoinForm.tsx
import type { HikerRole, JoinRequest } from "@/types.ts";

import styles from '@/routes/form.module.css'
import {useState} from "react";
import {useEditMyRequestMutation, useJoinRequestMutation} from "@/queries/outing.ts";


interface JoinProps{
    outingId: string
    onClose: () => void
    initial?: JoinRequest
}

export function JoinForm({outingId, onClose, initial}:JoinProps){

    const [hikerRole, setHikerRole] = useState<HikerRole|null>(initial?.role??null)
    const [seats, setSeats] = useState(initial?.seats_offered??0)
    const [guests, setGuests] = useState(initial?.guests??0)
    const [note, setNote] = useState<string>(initial?.note??"")

    const joinReqMutation = useJoinRequestMutation(outingId)
    const editJoinReqMutation = useEditMyRequestMutation(outingId)
    const mutation = initial?editJoinReqMutation:joinReqMutation
    function handleSubmit(e:React.SubmitEvent){
        e.preventDefault()
        if(!hikerRole) return
        mutation.mutate({role:hikerRole, seats_offered:seats, guests:guests, note:initial?note:note||undefined}, {onSuccess: () => {onClose()}})
    }
    const askMore = !!initial && (
        (initial.role === 'driver' && hikerRole === 'rider') ||
        guests > initial.guests ||
        (hikerRole === 'driver' && seats < initial.seats_offered)
    )
    return(
        <form aria-label={initial?'Edit your request':'Join outing'} className={styles.form} onSubmit={handleSubmit}>
            <fieldset className={styles.fieldset}>
                <legend className={styles.legend}>Hiker role</legend>
                <div className={styles.radioRow}>
                    <label className={`${styles.radioButton} ${hikerRole==='rider' ? styles.radioButtonChecked: ''}`}>
                        <input
                            name={"role"}
                            type="radio"
                            value='rider'
                            checked={hikerRole==='rider'}
                            onChange={e => {setHikerRole(e.target.value as HikerRole); setSeats(0)}}
                        />
                        Rider
                    </label>
                    <label className={`${styles.radioButton} ${hikerRole==='driver' ? styles.radioButtonChecked: ''}`}>
                        <input
                            name={"role"}
                            type="radio"
                            value='driver'
                            checked={hikerRole==='driver'}
                            onChange={e => {setHikerRole(e.target.value as HikerRole); setSeats(0)}}
                        />
                        Driver
                    </label>
                </div>
            </fieldset>

            {hikerRole == 'driver' &&
            <label className={styles.label}>
                Seats offered
                <input
                    type="number"
                    min={0}
                    value={seats}
                    onChange={e => setSeats(Number(e.target.value))}
                />
                <p className={styles.hint}>Includes your seat — 4 means you + 3 riders</p>
            </label>
            }
            <label className={styles.label}>
                Guests
                <input
                    type="number"
                    min={0}
                    max={3}
                    value={guests}
                    onChange={e => setGuests(Number(e.target.value))}
                />
            </label>
            <label className={styles.label}>
                Note to the host (optional)
                <textarea
                    placeholder={'Anything the host should know about?'}
                    value={note}
                    onChange={e => setNote(e.target.value)}
                ></textarea>
            </label>
            {mutation.error && <p className={styles.error}>{mutation.error.message}</p>}
            {askMore&&<p role="status" className={styles.notice}>Saving this moves you to the back of the waiting list.</p>}
            <button
               className={"btn-primary"}
                type={'submit'}
                disabled={!hikerRole || mutation.isPending}
            >{initial? 'Save changes':'Request to join'}</button>
            {!hikerRole && <p className={styles.hint}>Pick a role first.</p>}
            <button
                className={styles.quietBtn}
                type='button'
                onClick={onClose}
            >Never mind</button>
        </form>
    )


}