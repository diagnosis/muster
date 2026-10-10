import {createFileRoute, Link, useNavigate} from '@tanstack/react-router'
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {Detail, Member} from "@/types.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {useMeQuery, useOuting} from "@/queries.ts";
import {useEffect, useState} from "react";
import {JoinForm} from "@/components/JoinForm.tsx";
import {HostControls} from "@/components/HostControls.tsx";
import styles from "@/routes/outings.$id.module.css"
import formStyles from "@/routes/form.module.css"
import {Badges} from "@/components/Badges.tsx";
import {Modal} from "@/components/Modal.tsx";
import {Comments} from "@/components/Comments.tsx";
import {useEvents} from "@/events/EventProvider.tsx";
import {useStartDM} from "@/queries/conversation.ts";
import {CalendarIcon, FlagIcon, MessageIcon, PinIcon, TrashIcon} from "@/components/Icons.tsx";
import {outingRange} from "@/utils/date.ts";


export const Route = createFileRoute('/outings/$id')({
  component: OutingDetailPage,
})

export function OutingDetailPage() {
    const {id} = Route.useParams()
    const navigate = useNavigate()
    const qc = useQueryClient()
    const [showForm, setShowForm] = useState(false)
    const {data: me} = useMeQuery()
    const {data:detail, isPending, error} = useOuting(id)
    const startDM = useStartDM()
    const [memberToRemove, setMemberToRemove] = useState<Member|null>(null)
    const { subscribe } = useEvents()
    useEffect(()=> {
        return subscribe('notification.created', (data)=>{
            const poke = JSON.parse(data)
            if(poke.outing_id === id){
                qc.invalidateQueries({ queryKey: ['outings'] })
                qc.invalidateQueries({ queryKey: ['outing', id] })
                qc.invalidateQueries({ queryKey: ['my-outings'] })
                qc.invalidateQueries({ queryKey: ['outing-join-requests', id] })
            }
        })
    }, [subscribe, qc, id])

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
    const removeMemberMutation = useMutation({
        mutationFn: async(requestId: string) => {
            const res = await apiClient.del(`/api/requests/${requestId}/member`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () =>{
            qc.invalidateQueries({queryKey: ['outing', id]})
            qc.invalidateQueries({queryKey: ['my-outings']})
            setMemberToRemove(null)
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
            return rosterLocked
                ? <div className={styles.slot}><p>This outing has already started.</p></div>
                : <div className={`${styles.slot} ${styles.slotActive}`}>
                    <Link className={'btn-primary btn'} to="/login">Request to join</Link>
                </div>

        if (me.id === detail.outing.host_id) return <div className={`${styles.slot} ${styles.slotColumn}`}>
            <div className={styles.slotHead}>
                <p>You're hosting</p>
                {chatLink}
            </div>
            <HostControls outingId={id} detail={detail}/>
        </div>

        const st = detail.my_request?.status
        if (st === 'requested')
            return <div className={styles.slot}>
                <p>Requested — waiting on host</p>
                {!rosterLocked && <button onClick={() => withdrawMutation.mutate()}>Withdraw</button>}
            </div>

        if (st === 'accepted')
            return <div className={`${styles.slot} ${styles.slotActive}`}>
                <p>You're going! 🎉</p>
                <div className={styles.slotActions}>
                    {chatLink}
                    {!rosterLocked && <button onClick={() => withdrawMutation.mutate()}>Withdraw</button>}
                </div>
            </div>

        if (st === 'declined')
            return <div className={styles.slot}>
                <p>The host declined this request.</p>
            </div>
        if (st === 'removed')
            return <div className={styles.slot}>
                <p>The host removed you from this outing.</p>
            </div>
        if (rosterLocked) return <div className={styles.slot}>
            <p>This outing has already started.</p>
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

    const effectiveCap = Math.min(detail.seat_capacity, detail.outing.max_size)
    const isFull = detail.people_count >= detail.outing.max_size
    const canSeeComments = !!me && (
        me.id === detail.outing.host_id ||
            detail.my_request?.status === 'accepted' ||
            detail.my_request?.status === 'requested'
    )
    const canComment = detail.outing.status !== 'cancelled' && detail.outing.is_discussion_open
    const canSeeStartDmBtn = me && (me.id === detail.host.hiker_id || detail.roster.some(r => r.hiker_id === me.id))
    const canSeeChat =canSeeStartDmBtn && !!detail.outing.conversation_id
    const isHost = me?.id === detail.outing.host_id
    const initial = (name: string) => name.charAt(0).toUpperCase()
    const totalSpots = detail.people_count + detail.spots_left
    const filledPct = totalSpots > 0 ? Math.round((detail.people_count / totalSpots) * 100) : 0
    const chatLink = canSeeChat &&
        <Link className={'btn btn-primary'} to={'/conversations/$id'} params={{id: detail.outing.conversation_id}}>Outing chat</Link>
    const rosterLocked = detail.outing.phase !== 'upcoming'
    const messageBtn = (hikerId: string, name: string) => (
        <button className="icon-btn"
                aria-label={`Message ${name}`}
                disabled={startDM.isPending}
                onClick={() => startDM.mutate(hikerId, {
                    onSuccess: (conv) => navigate({to: '/conversations/$id', params: {id: conv.id}})
                })}>
            <MessageIcon/>
        </button>
    )
    return (
        <div className={styles.container}>
            <section className={styles.section}>
                <h1 className={styles.heading}>{detail.outing.title}</h1>
                <ul className={styles.meta}>
                    <li className={`${styles.metaItem} ${styles.metaStrong}`}><CalendarIcon/> {outingRange(detail.outing.starts_at, detail.outing.ends_at)}</li>
                    <li className={styles.metaItem}><PinIcon/> {detail.outing.destination}</li>
                    <li className={styles.metaItem}><FlagIcon/> {detail.outing.meet_label}</li>
                </ul>
                <Badges outing={detail.outing}/>
            </section>

            <section className={styles.section}>
                {renderSlot(detail)}
            </section>

            <section className={styles.section}>
                <p>{detail.people_count} going · {detail.spots_left} of {effectiveCap} spots left</p>
                <div className={styles.bar} aria-hidden="true">
                    <div className={styles.barFill} style={{width: `${filledPct}%`}}/>
                </div>
                {isFull && <p className={styles.warning}>This outing is full.</p>}
                {!isFull && detail.seats_short > 0 && <p>⚠️ {detail.seats_short} more seats needed — join as a driver?</p>}
                {!isFull && detail.seats_short === 0 && detail.spots_left === 0 && <p>No seats left — a driver could open more spots. 🚗</p>}
            </section>

            <section className={styles.section}>
                <h2 className={styles.subheading}>Who's going ({detail.roster.length + 1})</h2>
                <div className={styles.card}>
                    <div className={styles.memberRow}>
                        <span className={styles.avatar} aria-hidden="true">{initial(detail.host.name)}</span>
                        <p className={styles.hostRow}>{detail.host.name} · {detail.host.experience}</p>
                        <span className={styles.hostTag}>Host</span>
                        <span className={styles.rowActions}>
                        {canSeeStartDmBtn && detail.host.hiker_id !== me?.id && messageBtn(detail.host.hiker_id, detail.host.name)}
                    </span>
                    </div>
                    {detail.roster.map(m => <div className={styles.memberRow} key={m.hiker_id}>
                        <span className={styles.avatar} aria-hidden="true">{initial(m.name)}</span>
                        {m.name} · {m.experience}
                        <span className={styles.rowActions}>
                        {canSeeStartDmBtn && m.hiker_id !== me?.id && messageBtn(m.hiker_id, m.name)}
                            {isHost && !rosterLocked &&
                                <button aria-label={'Remove'} className={`icon-btn ${styles.removeIcon}`}
                                        onClick={() => setMemberToRemove(m)}><TrashIcon/></button>}
                    </span>
                    </div>)}
                </div>
                {startDM.isError && <p className={styles.error}>{startDM.error.message}</p>}
                {memberToRemove && <Modal title={`Remove ${memberToRemove.name}`} onClose={() => setMemberToRemove(null)}>
                    <p>This removes them from the roster and frees their seats. They won't be able to request again.</p>
                    <div className={styles.actions}>
                        <button className="btn-danger"
                                onClick={() => memberToRemove.request_id && removeMemberMutation.mutate(memberToRemove.request_id)}>Yes, remove</button>
                        <button className={formStyles.quietBtn}
                                onClick={() => setMemberToRemove(null)}>Never mind</button>
                    </div>
                    {removeMemberMutation.error && <p className={formStyles.error}>{removeMemberMutation.error.message}</p>}
                </Modal>}
            </section>

            {detail.outing.notes && <section className={styles.section}>
                <h2 className={styles.subheading}>Notes</h2>
                <p>{detail.outing.notes}</p>
            </section>}

            {canSeeComments && (
                <Comments outingId={id} hostId={detail.outing.host_id} canComment={canComment}/>
            )}
        </div>
    )
}
