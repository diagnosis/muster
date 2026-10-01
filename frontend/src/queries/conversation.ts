import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {Conversation, ListConversationsResponse} from "@/types.ts";
import {queryOptions, useMutation, useQuery, useQueryClient} from "@tanstack/react-query";


export async function listConversations(){
    const res = await apiClient.get<ListConversationsResponse>('/api/conversations')
    if (res.ok){
        return res.data
    }
    throw new ApiRequestError(res.error, res.httpStatus)

}

export const conversationsQueryOptions = () =>
    queryOptions({queryKey:['conversations'], queryFn: listConversations})

export function useConversations() {
    return useQuery(conversationsQueryOptions())
}

export function useStartDM(){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async (hiker_id:string) => {
            const res = await apiClient.post<Conversation>('/api/dms', {hiker_id})
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess : () => {
            qc.invalidateQueries({queryKey: ['conversations']})
        }
    })
}

export function useAcceptDM(cid:string) {
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async () => {
            const res = await apiClient.post(`/api/conversations/${cid}/accept`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['conversations'] })
            qc.invalidateQueries({ queryKey: ['conversation', cid] })
        }
    })
}

export function useDeclineDM(cid:string) {
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async () => {
            const res = await  apiClient.post(`/api/conversations/${cid}/decline`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            qc.invalidateQueries({queryKey:['conversations']})
            qc.invalidateQueries({queryKey:['conversation', cid]})
    }
    })
}

export function useReopenDM(cid:string){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async () => {
            const res = await apiClient.post(`/api/conversations/${cid}/reopen`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            qc.invalidateQueries({queryKey:['conversations']})
            qc.invalidateQueries({queryKey:['conversation', cid]})
        }
    })
}
