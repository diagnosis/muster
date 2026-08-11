import {createFileRoute, Link} from '@tanstack/react-router'
import {apiClient} from "../lib/api.ts";
import type {Detail} from "../types.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {useMeQuery, useOuting} from "../queries.ts";
import {useState} from "react";
import {JoinForm} from "../components/JoinForm.tsx";
import {HostControls} from "../components/HostControls.tsx";
import styles from "./outings.$id.module.css"

export const Route = createFileRoute('/outings/$id')({
  component: OutingDetailPage,
})

function OutingDetailPage() {
    const {id} = Route.useParams()
    const qc = useQueryClient()
    const [showForm, setShowForm] = useState(false)
    const {data: me} = useMeQuery()
    const {data:detail, isPending, error} = useOuting(id)


    const withdrawMutation = useMutation({
        mutationFn: async () => {
            const res = await apiClient.del(`/api/outings/${id}/requests/me`)
            if (res.ok){
                return res.data
            }
            throw new Error(res.error.message)
        },
        onSuccess : () => {
            qc.invalidateQueries({queryKey:['outing', id]})
        }
    })
    function handleOnClose(){
        setShowForm(false)
    }

    function renderSlot(detail: Detail){
        if (detail.outing.status === 'cancelled')
            return <div className={styles.slot}>
                <p>This outing was cancelled</p>
            </div>
        if (!me)
        return <div className={`${styles.slot} ${styles.slotActive}`}>
            <Link to="/login"><button className={'btn-primary'}>Request To join</button></Link>
        </div>

        if (me.id === detail.outing.host_id) return <div className={styles.slot}>
            <HostControls outingId={id} detail={detail}/>
        </div>

        const st = detail.my_request?.status
        if (st === 'requested')
            return <div className={styles.slot}>
                <p>Requested — waiting on host</p>
                <button onClick={() => withdrawMutation.mutate()}>Withdraw</button>
            </div>

        if (st === 'accepted')
            return <div className={`${styles.slot} ${styles.slotActive}`}>
                <p>You're going! 🎉</p>
                <button onClick={() => withdrawMutation.mutate()}>Withdraw</button>
            </div>

        if (st === 'declined')
            return <div className={styles.slot}>
                <p>The host declined this request.</p>
            </div>

        return <>
            {showForm
                ? <JoinForm outingId={id} onClose={handleOnClose}/>
                : <button onClick={() => setShowForm(true)}>Request to join</button>}
        </>
    }


    // Handle loading and error states before rendering
    if (isPending) return <div>Loading...</div>
    if (error) return <div>Error: {error.message}</div>
    const date = new Date(detail.outing.starts_at)
    const starts_at_date =
        date.toLocaleDateString('en-US', {
            weekday: 'short',
            month:'short',
            day:'numeric'
        })
    const starts_at_time=
        date.toLocaleTimeString('en-US', {
            hour:'numeric',
            minute:'2-digit'
        })

    const effectiveCap = Math.min(detail.seat_capacity, detail.outing.max_size)
    const isFull = detail.people_count >= detail.outing.max_size
    const seat_cost = `$${(detail.outing.cost_per_seat_cents / 100).toFixed(2)}/seat `


    return <>
        <div className={styles.container}>
            <section className={styles.section}>
                <h1 className={styles.heading}>{detail.outing.title}</h1>
                <p>{detail.outing.destination} · {detail.outing.meet_label} · {starts_at_date} · {starts_at_time}
                    {detail.outing.cost_per_seat_cents > 0 && ` · ${seat_cost}`}
                    </p>
            </section>

            <section className={styles.section}>
                {renderSlot(detail)}
            </section>
            <section className={styles.section}>
                <p>{detail.people_count} going · {detail.spots_left} of {effectiveCap} spots left</p>
                {isFull && <p>This outing is full.</p>}
                {!isFull && detail.seats_short > 0 && <p>⚠️ {detail.seats_short} more seats needed — join as a driver?</p>}
                {!isFull && detail.seats_short === 0 && detail.spots_left === 0 && <p>No seats left — a driver could open more spots. 🚗</p>}
            </section>

            <section className={styles.section}>
                <h3 className={styles.subheading}>Host</h3>
                <p>{detail.host.name} · {detail.host.experience}</p>
            </section>


            <section className={styles.section}>
                <h3 className={styles.subheading}>Who's going ({detail.roster.length})</h3>
                {detail.roster.length === 0
                    ? <p>No members yet — be the first.</p>
                    : detail.roster.map(m => <p key={m.hiker_id}>{m.name} · {m.experience}</p>)}

                {detail.outing.notes && <p>{detail.outing.notes}</p>}
            </section>

        </div>
    </>
}
