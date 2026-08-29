// src/components/OutingFrom.tsx

import type {CreateOutingInput, Difficulty, Outing, Pace} from "@/types.ts";
import styles from "@/routes/form.module.css"
import {type ApiRequestError} from "@/lib/api.ts";
import {useState} from "react";
import layout from "@/routes/outings.new.module.css"
import {isoToLocalInput} from "@/utils/date.ts";
import {parseNonNegativeFloat, parseNonNegativeInt, parsePositiveInt} from "@/utils/parser.ts";

interface OutingFormProps {
    heading: string
    initial? : Outing
    onSubmit: (values: CreateOutingInput) => void
    submitLabel: string,
    pending?: boolean,
    error?: ApiRequestError
}



export function OutingForm(props: OutingFormProps){
    const [title, setTitle] = useState(props.initial?.title ?? "")
    const [destination, setDestination] = useState(props.initial?.destination ?? "")
    const [meet_label, setMeet_label] = useState(props.initial?.meet_label ?? "")
    const [startsAt, setStartsAt] = useState(props.initial ? isoToLocalInput(props.initial.starts_at) : "")
    const [max_size, setMax_size] = useState<string>(props.initial?.max_size? `${props.initial.max_size}` : '2')
    const [host_seats, setHost_seats] = useState<string>(props.initial ? String(props.initial.host_seats) : '')
    const [costDollar, setCostDollar] = useState<string>(props.initial ? `${props.initial.cost_per_seat_cents / 100}` : '')
    const [difficulty, setDifficulty] = useState<Difficulty|null>(props.initial?.difficulty ?? null)
    const [pace, setPace] = useState<Pace|null>(props.initial?.pace ?? null)
    const [notes, setNotes] = useState<string>(props.initial?.notes ?? "")
    const [formError, setFormError] = useState<string|null>(null)

    function handleSubmit(e: React.SubmitEvent){
        e.preventDefault()
        const parsedMaxSize = parsePositiveInt(max_size)
        const parsedHostSeats = host_seats.trim() === "" ? 0 : parseNonNegativeInt(host_seats)
        const parsedCostDollar = costDollar.trim() ===""? 0 : parseNonNegativeFloat(costDollar, 2)
        if (!difficulty || !pace|| parsedMaxSize===null || parsedHostSeats===null || parsedCostDollar===null){
            setFormError("Check the number fields — something is invalid.")
            return;
        }

        setFormError(null)
        props.onSubmit({
            title, destination, meet_label,
            starts_at: new Date(startsAt).toISOString(),
            max_size:parsedMaxSize, host_seats:parsedHostSeats, cost_per_seat_cents: Math.round(parsedCostDollar*100),
            difficulty, pace,
            notes: notes || undefined,
        })
    }

    return <>
        <form className={`${styles.form} ${layout.createForm}`} onSubmit={handleSubmit}>
            <h1>{props.heading}</h1>
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
                        <label className={styles.label}>Meet label
                            <input
                                type="text"
                                value={meet_label}
                                onChange={e => setMeet_label(e.target.value)}
                            />
                        </label>
                        <label className={styles.label}>Starts at
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
                        <label className={styles.label}>Max size
                            <input
                                type="number"
                                min={2}
                                value={max_size}
                                onChange={e => setMax_size(e.target.value)}
                            />
                        </label>
                        <label className={styles.label}>Host seats
                            <input
                                type="number"
                                min={0}
                                value={host_seats}
                                onChange={e => setHost_seats(e.target.value)}
                            />
                            <span className={styles.hint}>Includes your seat — 4 means you + 3 riders</span>
                        </label>
                        <label className={styles.label}>Cost per seat
                            <input
                                type="number"
                                step={0.01}
                                min={0}
                                value={costDollar}
                                onChange={e => setCostDollar(e.target.value)}
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
            {props.error&&<p className={styles.error}>{props.error.message}</p>}
            {formError&&<p className={styles.error}>{formError}</p>}
            <button className={`btn-primary ${layout.submit}`} type={"submit"} disabled={!title || !destination ||!meet_label|| !startsAt || !difficulty || !pace || props.pending}>{props.submitLabel}</button>

        </form></>
}