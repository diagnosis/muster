
// src/components/JoinForm.tsx
import type {HikerRole, JoinRequest, JoinRequestInput} from "@/types.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import styles from '@/routes/form.module.css'
import {useState} from "react";


interface JoinProps{
    outingId: string
    onClose: () => void
}

export function JoinForm({outingId, onClose}:JoinProps){
    const queryClient = useQueryClient()
    const [hikerRole, setHikerRole] = useState<HikerRole|null>(null)
    const [seats, setSeats] = useState(0)
    const [guests, setGuests] = useState(0)
    const [note, setNote] = useState<string>("")

    const joinReqMutation = useMutation(
        {
            mutationFn : async (body: JoinRequestInput) => {
                const res =
                   await apiClient.post<JoinRequest>(`/api/outings/${outingId}/requests`, body)
                if (res.ok){
                    return res.data
                }
                throw new ApiRequestError(res.error, res.httpStatus)
            },
            onSuccess: () => {
                queryClient.invalidateQueries({queryKey:['outing', outingId]})
                onClose()
            }
        }
    )

    function handleSubmit(e:React.SubmitEvent){
        e.preventDefault()
        if(!hikerRole) return
        joinReqMutation.mutate({role:hikerRole, seats_offered:seats, guests:guests, note:note||undefined})
    }

    return(
        <form aria-label={'Join outing'} className={styles.form} onSubmit={handleSubmit}>
            <fieldset className={styles.fieldset}>
                <legend className={styles.legend}>Hiker Role</legend>
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
                Seats Offered
                <input
                    type="number"
                    min={0}
                    value={seats}
                    onChange={e => setSeats(Number(e.target.value))}
                />
                <p className={styles.hint}>Includes your seat. 4 means you + 3 riders</p>
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
                Note to the host (optional):
                <textarea
                    placeholder={'Anything the host should know about?'}
                    value={note}
                    onChange={e => setNote(e.target.value)}
                ></textarea>
            </label>
            {joinReqMutation.error && <p className={styles.error}>{joinReqMutation.error.message}</p>}

            <button
               className={"btn-primary"}
                type={'submit'}
                disabled={!hikerRole || joinReqMutation.isPending}
            >Request to join</button>
            {!hikerRole && <p className={styles.hint}>Pick a role first.</p>}
            <button
                className={styles.quietBtn}
                type='button'
                onClick={onClose}
            >Never mind</button>
        </form>
    )


}