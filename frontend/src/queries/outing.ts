import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {
    Detail,
    EditJoinRequestInput,
    JoinRequest, JoinRequestInput,
    JoinRequestView,
    MyOutings,
    Outing,
    RequestStatus
} from "@/types.ts";
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query";


export const getOutingByID = async (id:string) => {
    const res = await apiClient.get<Detail>(`/api/outings/${id}`)
    if (res.ok){
        return res.data
    }
    throw new ApiRequestError(res.error, res.httpStatus)
}

export const outingQueryOptions = (id: string) => ({
    queryKey: ['outing', id],
    queryFn: () => getOutingByID(id),
})

export function useOuting(id: string) {
    return useQuery(outingQueryOptions(id))
}

export const getOutings=async ()=>{
    const res = await apiClient.get<Outing[]>('/api/outings')
    if (res.ok){
        return res.data
    }
    throw new ApiRequestError(res.error, res.httpStatus)
}

export function useOutings(){
    return useQuery({
        queryKey:['outings'],
        queryFn:getOutings,
    })
}



export function useOutingJoinRequests(id: string, status:RequestStatus) {
    return useQuery(
        {
            queryKey:['outing-join-requests', id, status],
            queryFn: async () => {
                const res =
                    await apiClient.get<JoinRequestView[]>(`/api/outings/${id}/requests?status=${status}`)
                if(res.ok){
                    return res.data
                }
                throw new ApiRequestError(res.error, res.httpStatus)
            }

        }
    )
}


// my outings

export function useMyOutings(){
    return useQuery(
        {
            queryKey:['my-outings'],
            queryFn: async () => {
                const res = await apiClient.get<MyOutings>(`/api/me/outings`)
                if (res.ok){
                    return res.data
                }
                throw new ApiRequestError(res.error, res.httpStatus)
            }
        }
    )
}


// join requests


export function useJoinRequestMutation(outingId:string){
    const qc = useQueryClient()
    return useMutation(
        {
            mutationFn : async (body: JoinRequestInput) => {
                const res =
                    await apiClient.post<JoinRequest>(`/api/outings/${outingId}/requests`, body)
                if (res.ok){
                    return res.data
                }
                throw new ApiRequestError(res.error, res.httpStatus)
            },
            onSuccess: () => {
                qc.invalidateQueries({queryKey:['outing', outingId]})
                qc.invalidateQueries({queryKey:['my-outings']})
            }
        }
    )
}

export function useEditMyRequestMutation(outingId:string){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async (edit:EditJoinRequestInput) => {
            const res = await apiClient.patch<JoinRequest>(`/api/outings/${outingId}/requests/me`, edit)
            if (res.ok) {
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            qc.invalidateQueries({queryKey:['outing', outingId]})
            qc.invalidateQueries({queryKey:['outing-join-requests', outingId]})
            qc.invalidateQueries({queryKey:['my-outings']})
        }

    })
}

export function useWithdrawMutation(id:string){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async () => {
            const res = await apiClient.del(`/api/outings/${id}/requests/me`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess : () => {
            qc.invalidateQueries({queryKey:['outing', id]})
            qc.invalidateQueries({queryKey: ['my-outings']})
            qc.invalidateQueries({queryKey:['outing-join-requests', id]})
        }
    })
}

export function useRemoveMemberMutation(id:string){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async(requestId: string) => {
            const res = await apiClient.del(`/api/requests/${requestId}/member`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () =>{
            qc.invalidateQueries({queryKey:['outing', id]})
            qc.invalidateQueries({queryKey: ['my-outings']})
            qc.invalidateQueries({queryKey:['outing-join-requests', id]})
        }
    })
}