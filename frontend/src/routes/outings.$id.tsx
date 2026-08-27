import {createFileRoute, Link} from '@tanstack/react-router'
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {Detail} from "@/types.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {useMeQuery, useOuting} from "@/queries.ts";
import {useState} from "react";
import {JoinForm} from "@/components/JoinForm.tsx";
import {HostControls} from "@/components/HostControls.tsx";
import styles from "@/routes/outings.$id.module.css"
import {Badges} from "@/components/Badges.tsx";

export const Route = createFileRoute('/outings/$id')({
  component: OutingDetailPage,
})

export function OutingDetailPage() {
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
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess : () => {
            qc.invalidateQueries({queryKey: ['outing', id]})
            qc.invalidateQueries({queryKey: ['my-outings']})
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
            <Link className={'btn-primary btn'} to="/login">Request to join</Link>
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
                ? <>
                    {isFull && <p className={styles.warning}>⚠️This outing is full — you can still request in case a spot opens.</p>}
                    <JoinForm outingId={id} onClose={handleOnClose}/>
                </>
                : <>
                    {isFull && <p className={styles.warning}>⚠️This outing is full — you can still request in case a spot opens.</p>}
                    <button className={'btn btn-primary'} onClick={() => setShowForm(true)}>Request to join</button>
                </>}
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


    return <>
        <div className={styles.container}>
            <section className={styles.section}>
                <h1 className={styles.heading}>{detail.outing.title}</h1>
                <p className={styles.metaLine}>{detail.outing.destination}</p>
                <p className={styles.metaLine}>{detail.outing.meet_label}</p>
                <p className={styles.metaLine}>{starts_at_date} - {starts_at_time}</p>
                <Badges outing={detail.outing}/>

            </section>

            <section className={styles.section}>
                {renderSlot(detail)}
            </section>
            <section className={styles.section}>
                <p>{detail.people_count} going · {detail.spots_left} of {effectiveCap} spots left</p>
                {isFull && <p className={styles.warning}>This outing is full.</p>}
                {!isFull && detail.seats_short > 0 && <p>⚠️ {detail.seats_short} more seats needed — join as a driver?</p>}
                {!isFull && detail.seats_short === 0 && detail.spots_left === 0 && <p>No seats left — a driver could open more spots. 🚗</p>}
            </section>



            <section className={styles.section}>
                <h3 className={styles.subheading}>Who's going ({detail.roster.length + 1})</h3>
                <p className={styles.hostRow}>{detail.host.name} · {detail.host.experience} · host</p>
                {detail.roster.map(m => <p key={m.hiker_id}>{m.name} · {m.experience}</p>)}
            </section>
            {detail.outing.notes&&<section className={styles.section}>
                <h3 className={styles.subheading}>Notes</h3>
                {detail.outing.notes && <p>{detail.outing.notes}</p>}
            </section>}

        </div>
    </>
}
