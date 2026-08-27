
//src/components/HostControls.tsx
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {Detail, JoinRequest, Outing, PendingRequestResponse} from "@/types.ts";
import {useOutingJoinRequests} from "@/queries.ts";
import {useState} from "react";
import {Modal} from "@/components/Modal.tsx";
import styles from "@/components/HostControls.module.css"
import {Link} from "@tanstack/react-router";

interface HostControlsProps{
    outingId: string
    detail: Detail
}
export function HostControls( {outingId, detail}: HostControlsProps ){
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


    function handleCancel(){

        if (!window.confirm("Cancel this outing? Members will see it as cancelled. this can't be undone.")) return
        cancelMutation.mutate()
    }
    const requestLen = requests?.length
    const isDriver =  selectedRequest?.role == "driver"
    const wontFit = !isDriver && 1 + Number(selectedRequest?.guests) > detail.spots_left

    return <div className={styles.host}>
            {requestsError&&<p>{requestsError.message}</p>}
            {requestsPending&&<p>requests loading...</p>}
        <h2 className={styles.subheading}>Requests ({requestLen ?? 0})</h2>
        {requests?.map(r => (
            <button
                className={styles.requestRow}
                onClick={()=>setSelectedRequest(r)}
                key={r.id}>
                {r.hiker_name} requests as {r.role}
                {r.guests > 0 && ` and brings ${r.guests} guest${r.guests>1?'s':''}`}
            </button>
        ))}
        {selectedRequest&&(
            <Modal title={`${selectedRequest.hiker_name} · ${selectedRequest.hiker_experience}`} onClose={() => setSelectedRequest(null)}>
                <p>
                    {selectedRequest.role}
                    {selectedRequest.seats_offered > 0 && ` · offering ${selectedRequest.seats_offered} seat${selectedRequest.seats_offered > 1 ? 's' : ''}`}
                    {selectedRequest.guests > 0 && ` · +${selectedRequest.guests} ${selectedRequest.guests===1?'guest':'guests'}`}
                </p>

                {selectedRequest.note && <p>{selectedRequest.note}</p>}

                <p>Accepting adds {1 + selectedRequest.guests} {selectedRequest.guests>0?"people":"person"}
                    </p>

                {wontFit && <p>⚠️ Looks like there isn't room for {1 + selectedRequest.guests} right now.</p>}

                <div className={styles.actions}>
                    <button className="btn-primary" onClick={() => acceptMutation.mutate(selectedRequest.id)}>Accept</button>
                    <button className={styles.declineBtn} onClick={() => declineMutation.mutate(selectedRequest.id)}>Decline</button>
                </div>
                {acceptMutation.error && <p className={styles.error}>{acceptMutation.error.message}</p>}
                {declineMutation.error && <p className={styles.error}>{declineMutation.error.message}</p>}
            </Modal>
        )}

        <div className={styles.actions}>
            <Link className={'btn'} to="/outings/$id/edit" params={{id: outingId}}>Edit outing</Link>
            <button className={styles.cancelBtn} onClick={handleCancel}>Cancel outing</button>
        </div>
        </div>



}