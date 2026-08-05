
import {apiClient} from "../lib/api.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {Outing} from "../types.ts";
import {useOutingJoinRequests} from "../queries.ts";

interface HostControlsProps{
    outingId: string
}
export function HostControls( {outingId}: HostControlsProps ){
    const qc= useQueryClient()
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
            <p key={r.id}>
                {r.hiker_name} · {r.role}
                {r.guests > 0 && ` · +${r.guests} guest`}
                {r.note && ` — "${r.note}"`}
            </p>
        ))}

        <button onClick={handleCancel}>Cancel Outing</button>

    </>


}