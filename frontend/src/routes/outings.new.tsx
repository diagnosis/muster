import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {requireAuth} from "../queries.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {useState} from "react";
import type {CreateOutingInput, Difficulty, Outing, Pace} from "../types.ts";
import styles from "./form.module.css"
import layout from "./outings.new.module.css"
import {apiClient} from "../lib/api.ts";

export const Route = createFileRoute('/outings/new')({
    beforeLoad : async ({context}) => requireAuth(context.queryClient),
  component: CreateOutingPage,
})

function CreateOutingPage() {

    const qc = useQueryClient()
    const navigate = useNavigate()
    const [title, setTitle] = useState("")
    const [destination, setDestination] = useState("")
    const [meet_label, setMeet_label] = useState("")
    const [startsAt, setStartsAt] = useState("")
    const [max_size, setMax_size] = useState(2)
    const [host_seats, setHost_seats] = useState(0)
    const [costDollar, setCostDollar] = useState(0)
    const [difficulty, setDifficulty] = useState<Difficulty|null>(null)
    const [pace, setPace] = useState<Pace|null>(null)
    const [notes, setNotes] = useState<string>("")

    const createMutation = useMutation({
        mutationFn: async (body:CreateOutingInput) => {
            const res = await apiClient.post<Outing>('/api/outings', body)
            if (res.ok){
                return res.data
            }
            throw new Error(res.error.message)
        },
        onSuccess:(data)=>{
            qc.invalidateQueries({queryKey:['outings']})
            navigate({to:'/outings/$id', params: {id: data.id}})
        }
    })

    function handleSubmit(e: React.SubmitEvent){
        e.preventDefault()
        if (!difficulty || !pace) return
        createMutation.mutate({
            title, destination, meet_label,
            starts_at: new Date(startsAt).toISOString(),
            max_size, host_seats, cost_per_seat_cents: Math.round(costDollar*100),
            difficulty, pace,
            notes: notes || undefined,
        })
    }

  return (
      <div>
          <form className={`${styles.form} ${layout.createForm}`} onSubmit={handleSubmit}>
            <h1>Create Outing</h1>
            <div className={layout.groups}>
              <div className={layout.group}>
                <div className={layout.groupLabel}>Where &amp; when</div>
                <div className={layout.fields}>
              <label className={styles.label}>Title
                  <input
                      type="text"
                      value={title}
                      onChange={e => setTitle(e.target.value)}
                  />
              </label>
              <label className={styles.label}>Destination
                  <input
                      type="text"
                      value={destination}
                      onChange={e => setDestination(e.target.value)}
                  />
              </label>
              <label className={styles.label}>Meet Label
                  <input
                      type="text"
                      value={meet_label}
                      onChange={e => setMeet_label(e.target.value)}
                  />
              </label>
              <label className={styles.label}>Starts At
                  <input
                      type="datetime-local"
                      value={startsAt}
                      onChange={e => setStartsAt(e.target.value)}
                  />
              </label>
                </div>
              </div>

              <div className={layout.group}>
                <div className={layout.groupLabel}>Seats &amp; cost</div>
                <div className={layout.fields}>
              <label className={styles.label}>Max Size
                  <input
                      type="number"
                      min={2}
                      value={max_size}
                      onChange={e => setMax_size(Number(e.target.value))}
                  />
              </label>
              <label className={styles.label}>Host Seats
                  <input
                      type="number"
                      min={0}
                      value={host_seats}
                      onChange={e => setHost_seats(Number(e.target.value))}
                  />
              </label>
              <label className={styles.label}>Cost per Seat
                    <input
                        type="number"
                        min={0}
                        value={costDollar}
                        onChange={e => setCostDollar(Number(e.target.value))}
                    />
              </label>
                </div>
              </div>

              <div className={layout.group}>
                <div className={layout.groupLabel}>What it's like</div>
                <div className={layout.fields}>

                    <fieldset className={styles.fieldset}>
                        <legend className={styles.legend}>Difficulty</legend>
                        <div className={styles.radioRow}>
                            <label className={`${styles.radioButton} ${difficulty === 'easy'? styles.radioButtonChecked:""}`}>
                                <input
                                    name={"difficulty"}
                                    type="radio"
                                    value="easy"
                                    checked={difficulty==="easy"}
                                    onChange={e=> setDifficulty(e.target.value as Difficulty)}
                                />
                                Easy
                            </label>
                            <label className={`${styles.radioButton} ${difficulty === 'moderate'? styles.radioButtonChecked:""}`}>
                                <input
                                    name={"difficulty"}
                                    type="radio"
                                    value="moderate"
                                    checked={difficulty==="moderate"}
                                    onChange={e=> setDifficulty(e.target.value as Difficulty)}
                                />
                                Moderate
                            </label>
                            <label className={`${styles.radioButton} ${difficulty === 'hard'? styles.radioButtonChecked:""}`}>
                                <input
                                    name={"difficulty"}
                                    type="radio"
                                    value="hard"
                                    checked={difficulty==="hard"}
                                    onChange={e=> setDifficulty(e.target.value as Difficulty)}
                                />
                                Hard
                            </label>
                        </div>
                    </fieldset>
             <fieldset className={styles.fieldset}>
                 <legend className={styles.legend}>Pace</legend>
                 <div className={styles.radioRow}>
                     <label className={`${styles.radioButton} ${pace === "relaxed"? styles.radioButtonChecked:""}`}>
                         <input
                             name={"pace"}
                             type="radio"
                             value="relaxed"
                             checked={pace==="relaxed"}
                             onChange={e => setPace(e.target.value as Pace)}
                         />
                         Relaxed
                     </label>
                     <label className={`${styles.radioButton} ${pace==="moderate" ? styles.radioButtonChecked:''}`}>
                         <input
                             name={"pace"}
                             type="radio"
                             value="moderate"
                             checked={pace==="moderate"}
                             onChange={e => setPace(e.target.value as Pace)}
                         />
                         Moderate
                     </label>
                     <label className={`${styles.radioButton} ${pace==="fast" ? styles.radioButtonChecked:''}`}>
                         <input
                             name={"pace"}
                             type="radio"
                             value="fast"
                             checked={pace==="fast"}
                             onChange={e => setPace(e.target.value as Pace)}
                         />
                         Fast
                     </label>
                 </div>
             </fieldset>
              <label className={styles.label}>
                  Notes
                <textarea
                    placeholder={'Anything the members should know about?'}
                    value={notes}
                    onChange={e => setNotes(e.target.value)}
                ></textarea>
              </label>
                </div>
              </div>
            </div>
            <button className={`btn-primary ${layout.submit}`} type={"submit"} disabled={!title || !destination || !startsAt || !difficulty || !pace || createMutation.isPending}>Create Outing</button>
              {createMutation.error&&<p className={styles.error}>{createMutation.error.message}</p>}
          </form>
      </div>
  )
}
