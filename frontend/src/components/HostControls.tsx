
import {apiClient} from "../lib/api.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {Detail, JoinRequest, Outing, PendingRequestResponse} from "../types.ts";
import {useOutingJoinRequests} from "../queries.ts";
import {useState} from "react";
import {Modal} from "./Modal.tsx";

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

    return <>
        {requestsError&&<p>{requestsError.message}</p>}
        {requestsPending&&<p>requests loading...</p>}
        <h3>Requests ({requestLen ?? 0})</h3>
        {requests?.map(r => (
            <p
                onClick={()=>setSelectedRequest(r)}
                key={r.id}>
                {r.hiker_name} · {r.role}
                {r.guests > 0 && ` · +${r.guests} guest`}
                {r.note && ` — "${r.note}"`}
            </p>
        ))}
        {selectedRequest&&(
            <Modal onClose={()=> setSelectedRequest(null)}>
                <button onClick={()=> setSelectedRequest(null)}>close</button>
                <h3>{selectedRequest.hiker_name} {"·"} {selectedRequest.hiker_experience}</h3>
                <p>{selectedRequest.role}
                    {selectedRequest.seats_offered>0 &&` · offering ${selectedRequest.seats_offered}`}
                    {selectedRequest.guests > 0 && ` · +${selectedRequest.guests} guest`}
                    {selectedRequest.note && <p>{selectedRequest.note}</p>}
                    <p>Accepting adds {1+ selectedRequest.guests} - {detail.spots_left} spots left</p>
                    {1+selectedRequest.guests > detail.spots_left && (
                        <p>⚠️Doesn't fit right now - more seats or spots needed</p>
                    )}
                    <button onClick={ () => acceptMutation.mutate(selectedRequest.id)}>Accept</button>
                    <p>{acceptMutation.error&&acceptMutation.error.message}</p>
                    <button onClick={ () => declineMutation.mutate(selectedRequest.id)}>Decline</button>
                    <p>{declineMutation.error&&declineMutation.error.message}</p>

                </p>
            </Modal>
        )}

        <h2>Declined Request </h2>
        <button onClick={handleCancel}>Cancel Outing</button>

    </>


}