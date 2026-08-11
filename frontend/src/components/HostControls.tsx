
//src/components/HostControls.tsx
import {apiClient} from "../lib/api.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {Detail, JoinRequest, Outing, PendingRequestResponse} from "../types.ts";
import {useOutingJoinRequests} from "../queries.ts";
import {useState} from "react";
import {Modal} from "./Modal.tsx";
import styles from "./HostControls.module.css"

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
            throw new Error(res.error.message)
        },
        onSuccess: () => {
            qc.invalidateQueries({queryKey: ['outing', outingId]})
            qc.invalidateQueries({queryKey:['outings']})
        },
    })

    const acceptMutation =  useMutation({
        mutationFn: async (requestId:string) => {
            const res = await apiClient.post<JoinRequest>(`/api/requests/${requestId}/accept`)
            if (res.ok){
                return res.data
            }
            throw new Error(res.error.message)
        },
        onSuccess: () => {
            qc.invalidateQueries({queryKey:['outing', outingId]})
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
            throw new Error(res.error.message)
        }, onSuccess:() => {
            qc.invalidateQueries({queryKey:['outing', outingId]})
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
        <h3>Requests ({requestLen ?? 0})</h3>
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
            <Modal onClose={() => setSelectedRequest(null)}>
                <div className={styles.modalHeader}>
                    <h3>{selectedRequest.hiker_name} · {selectedRequest.hiker_experience}</h3>
                    <button className={styles.closeBtn} onClick={() => setSelectedRequest(null)}>✕</button>
                </div>
                <p>
                    {selectedRequest.role}
                    {selectedRequest.seats_offered > 0 && ` · offering ${selectedRequest.seats_offered}`}
                    {selectedRequest.guests > 0 && `  +${selectedRequest.guests} guest`}
                </p>

                {selectedRequest.note && <p>{selectedRequest.note}</p>}

                <p>Accepting adds {1 + selectedRequest.guests} {selectedRequest.guests>0?"people":"person"}
                    </p>

                {wontFit && <p>⚠️ Not enough seats — someone needs to drive</p>}

                <div className={styles.actions}>
                    <button disabled={wontFit} className="btn-primary" onClick={() => acceptMutation.mutate(selectedRequest.id)}>Accept</button>
                    <button className={styles.declineBtn} onClick={() => declineMutation.mutate(selectedRequest.id)}>Decline</button>
                </div>
                {acceptMutation.error && <p className={styles.error}>{acceptMutation.error.message}</p>}
                {declineMutation.error && <p className={styles.error}>{declineMutation.error.message}</p>}
            </Modal>
        )}

        <button className={styles.cancelBtn} onClick={handleCancel}>Cancel Outing</button>
        </div>



}