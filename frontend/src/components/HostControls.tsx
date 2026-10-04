
//src/components/HostControls.tsx
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {Detail, JoinRequest, Outing, PendingRequestResponse} from "@/types.ts";
import {useOutingJoinRequests} from "@/queries.ts";
import {useState} from "react";
import {Modal} from "@/components/Modal.tsx";
import styles from "@/components/HostControls.module.css"
import {Link, useNavigate} from "@tanstack/react-router";
import {useStartDM} from "@/queries/conversation.ts";
import {MessageIcon} from "@/components/Icons.tsx";

interface HostControlsProps{
    outingId: string
    detail: Detail
}
export function HostControls( {outingId, detail}: HostControlsProps ){
    const navigate = useNavigate()
    const qc= useQueryClient()
    const [selectedRequest, setSelectedRequest] = useState<PendingRequestResponse | null>(null)
    const { data: requests, isPending: requestsPending, error:requestsError} = useOutingJoinRequests(outingId)


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

    const startDM = useStartDM()


    function handleCancel(){

        if (!window.confirm("Cancel this outing? Members will see it as cancelled. this can't be undone.")) return
        cancelMutation.mutate()
    }
    const requestLen = requests?.length
    const isDriver =  selectedRequest?.role == "driver"
    const wontFit = !isDriver && 1 + Number(selectedRequest?.guests) > detail.spots_left

    return <div className={styles.host}>
        {requestsError && <p className={styles.error}>{requestsError.message}</p>}
        {requestsPending && <p>requests loading...</p>}
        <h2 className={styles.subheading}>Requests ({requestLen ?? 0})</h2>
        {requests && requests.length > 0 &&
            <div className={styles.requestList}>
                {requests.map(r => (
                    <div key={r.id} className={styles.requestItem}>
                        <button className={styles.requestRow} onClick={() => setSelectedRequest(r)}>
                            <span className={styles.reqName}>{r.hiker_name}</span>{' '}requests as {r.role}
                            {r.guests > 0 && ` and brings ${r.guests} guest${r.guests > 1 ? 's' : ''}`}
                        </button>
                        <button className="icon-btn"
                                aria-label={`Message ${r.hiker_name}`}
                                onClick={() => {
                                    startDM.mutate(r.hiker_id, {
                                        onSuccess: (conv) => navigate({to: '/conversations/$id', params: {id: conv.id}})
                                    })
                                }}><MessageIcon/></button>
                    </div>
                ))}
            </div>}
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

        <div className={styles.manage}>
            <Link className={'btn'} to="/outings/$id/edit" params={{id: outingId}}>Edit outing</Link>
            <button className={'btn-danger'} onClick={handleCancel}>Cancel outing</button>
        </div>
        </div>



}