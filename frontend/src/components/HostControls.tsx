
//src/components/HostControls.tsx
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {Detail, JoinRequest, JoinRequestView, Outing,} from "@/types.ts";

import {useState} from "react";
import {Modal} from "@/components/Modal.tsx";
import styles from "@/components/HostControls.module.css"
import {Link, useNavigate} from "@tanstack/react-router";
import {useStartDM} from "@/queries/conversation.ts";
import {useOutingJoinRequests} from "@/queries/outing.ts";
import {MessageIcon} from "@/components/Icons.tsx";
import {relativeTime} from "@/utils/date.ts";

interface HostControlsProps{
    outingId: string
    detail: Detail
}
type TabStatus = 'requested'|'declined'|'removed'
export function HostControls( {outingId, detail}: HostControlsProps ){
    const navigate = useNavigate()
    const qc= useQueryClient()
    const [selectedRequest, setSelectedRequest] = useState<JoinRequestView | null>(null)
    const [status, setStatus] = useState<TabStatus>('requested')
    const waiting  = useOutingJoinRequests(outingId, 'requested')
    const declined = useOutingJoinRequests(outingId, 'declined')
    const removed  = useOutingJoinRequests(outingId, 'removed')
    const rosterLocked = detail.outing.phase !== 'upcoming'

    const cancelMutation = useMutation({
        mutationFn: async  () => {
            const res = await apiClient.post<Outing>(`/api/outings/${outingId}/cancel`)
            if(res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            qc.invalidateQueries({queryKey: ['outing', outingId]})
            qc.invalidateQueries({queryKey:['outings']})
            qc.invalidateQueries({queryKey: ['my-outings']})
        },
    })

    const acceptMutation =  useMutation({
        mutationFn: async (requestId:string) => {
            const res = await apiClient.post<JoinRequest>(`/api/requests/${requestId}/accept`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            qc.invalidateQueries({queryKey:['outing', outingId]})
            qc.invalidateQueries({queryKey: ['my-outings']})
            qc.invalidateQueries({queryKey:['outing-join-requests', outingId]})
            setSelectedRequest(null)
        }
    })

    const declineMutation = useMutation({
        mutationFn: async (requestId: string) => {
            const res = await apiClient.post<JoinRequest>(`/api/requests/${requestId}/decline`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        }, onSuccess:() => {
            qc.invalidateQueries({queryKey:['outing', outingId]})
            qc.invalidateQueries({queryKey: ['my-outings']})
            qc.invalidateQueries({queryKey:['outing-join-requests', outingId]})
            setSelectedRequest(null)
        }
    })

    const restoreMutation = useMutation({
        mutationFn: async (requestId:string) => {
            const res = await apiClient.post(`/api/requests/${requestId}/restore`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        }, onSuccess:() => {
            qc.invalidateQueries({queryKey:['outing', outingId]})
            qc.invalidateQueries({queryKey: ['my-outings']})
            qc.invalidateQueries({queryKey:['outing-join-requests', outingId]})
        }
    })

    const startDM = useStartDM()


    function handleCancel(){

        if (!window.confirm("Cancel this outing? Members will see it as cancelled. this can't be undone.")) return
        cancelMutation.mutate()
    }

    const isDriver =  selectedRequest?.role == "driver"
    const wontFit = !isDriver && 1 + Number(selectedRequest?.guests) > detail.spots_left
    const labels: Record<TabStatus, string> = {
        requested: 'Waiting',
        declined: 'Declined',
        removed: 'Removed',
    }
    const counts: Record<TabStatus, number> = {
        requested: waiting.isSuccess ? waiting.data.length : 0,
        declined: declined.isSuccess ? declined.data.length : 0,
        removed: removed.isSuccess ? removed.data.length : 0,
    }
    const byStatus: Record<TabStatus, ReturnType<typeof useOutingJoinRequests>> = {
        requested: waiting,
        declined: declined,
        removed: removed,
    }
    const active = byStatus[status]
    const { data: requests, isPending: requestsPending, error: requestsError } = active

    const emptyCopy: Record<TabStatus, string> = {
        requested: 'Nobody is waiting.',
        declined: "You haven't declined anyone.",
        removed: "You haven't removed anyone.",
    }

    return <div className={styles.host}>
        {requestsError && <p className={styles.error}>{requestsError.message}</p>}
        {requestsPending && <p>requests loading...</p>}
        {rosterLocked
            ? requests && requests.length > 0 &&
            <div className={styles.lockedNote}>
                <p className={styles.lockedTitle}>
                    {requests.length} request{requests.length > 1 ? 's were' : ' was'} still waiting when this started
                </p>
                <p className={styles.lockedText}>They can't be accepted or declined any more.</p>
            </div>
            : <>
                <h2 className={styles.subheading}>Requests</h2>
                <div className={styles.filter} role="tablist" aria-label="Filter requests by status">
                    {(['requested', 'declined', 'removed'] as const).map(s => (
                        <button key={s} type="button" role="tab"
                                aria-selected={status === s}
                                className={status === s ? styles.tabActive : styles.tab}
                                onClick={() => setStatus(s)}>
                            {labels[s]} ({counts[s] ?? 0})
                        </button>
                    ))}
                </div>

                {requests && requests.length > 0 ?
                    <div className={styles.requestList}>
                        {requests.map(r => (
                            <div key={r.id} className={styles.requestItem}>
                                {status === 'requested'
                                    ? <button className={styles.requestRow} onClick={() => setSelectedRequest(r)}>
                                        <span className={styles.reqName}>{r.hiker_name}</span>{' '}requests as {r.role}
                                        {r.guests > 0 && ` and brings ${r.guests} guest${r.guests > 1 ? 's' : ''}`}
                                    </button>
                                    : <>
                                        <div className={styles.historyRow}>
                                            <p><span className={styles.reqName}>{r.hiker_name}</span>{' '}asked as {r.role}
                                                {r.guests > 0 && ` and brings ${r.guests} guest${r.guests > 1 ? 's' : ''}`}</p>
                                            <p className={styles.historyWhen}>{labels[status]} {relativeTime(r.updated_at)}</p>
                                        </div>
                                        <button className={styles.restoreBtn} onClick={() => restoreMutation.mutate(r.id)}
                                                disabled={restoreMutation.isPending}>Restore</button>
                                    </>}
                                <button className="icon-btn" aria-label={`Message ${r.hiker_name}`}  onClick={() => {
                                    startDM.mutate(r.hiker_id, {
                                        onSuccess: (conv) => navigate({to: '/conversations/$id', params: {id: conv.id}})
                                    })
                                }}><MessageIcon/></button>
                            </div>
                        ))}
                    </div>:<p className={styles.empty}>{emptyCopy[status]}</p>}
                {restoreMutation.error && <p className={styles.error}>{restoreMutation.error.message}</p>}
            </>
        }
        {selectedRequest&&(
            <Modal title={selectedRequest.hiker_name} onClose={() => setSelectedRequest(null)}>
                <dl className={styles.facts}>
                    <div className={styles.fact}><dt>Experience</dt><dd>{selectedRequest.hiker_experience}</dd></div>
                    <div className={styles.fact}><dt>Joining as</dt><dd>{selectedRequest.role}</dd></div>
                    {selectedRequest.seats_offered > 0 &&
                        <div className={styles.fact}><dt>Seats offered</dt><dd>{selectedRequest.seats_offered}</dd></div>}
                    {selectedRequest.guests > 0 &&
                        <div className={styles.fact}><dt>Guests</dt><dd>{selectedRequest.guests}</dd></div>}
                </dl>

                {selectedRequest.note &&
                    <div>
                        <p className={styles.noteLabel}>Note to host</p>
                        <p className={styles.note}>{selectedRequest.note}</p>
                    </div>}

                <p className={styles.impact}>Accepting adds {1 + selectedRequest.guests} {selectedRequest.guests > 0 ? "people" : "person"}</p>

                {wontFit && <p className={styles.warn}>⚠️ Looks like there isn't room for {1 + selectedRequest.guests} right now.</p>}

                <div className={styles.decision}>
                    <button className="btn-primary" onClick={() => acceptMutation.mutate(selectedRequest.id)}>Accept</button>
                    <button className="btn-danger" onClick={() => declineMutation.mutate(selectedRequest.id)}>Decline</button>
                </div>
                {acceptMutation.error && <p className={styles.error}>{acceptMutation.error.message}</p>}
                {declineMutation.error && <p className={styles.error}>{declineMutation.error.message}</p>}
            </Modal>
        )}

        {!rosterLocked && <div className={styles.manage}>
            <Link className={'btn'} to="/outings/$id/edit" params={{id: outingId}}>Edit outing</Link>
            <button className={'btn-danger'} onClick={handleCancel}>Cancel outing</button>
        </div>}
        </div>



}