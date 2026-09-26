// src/queries/message.ts


import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {ListMessagesResponse, Message, MessageInput} from "@/types.ts";
import {queryOptions, useMutation, useQuery, useQueryClient} from "@tanstack/react-query";

export async function listMessages(cid: string){
    const res = await apiClient.get<ListMessagesResponse>(`/api/conversations/${cid}/messages`)
    if (res.ok){
        return res.data
    }
    throw new ApiRequestError(res.error, res.httpStatus)
}

export const messagesQueryOptions = (cid:string)=>{
    return queryOptions({
        queryKey: ['messages', cid],
        queryFn: () => listMessages(cid)
    })
}

export function useListMessages(cid:string){
    return(
        useQuery(messagesQueryOptions(cid))
    )
}
export function useDeleteMessage(cid:string){
    const qc = useQueryClient()
    return useMutation(
        {
            mutationFn: async (mid: string) => {
                const res = await apiClient.del(`/api/messages/${mid}`)
                if (res.ok){
                    return res.data
                }
                throw new ApiRequestError(res.error, res.httpStatus)
            },
            onSuccess : () => qc.invalidateQueries({ queryKey: ['messages', cid]})
        }
    )
}

export function usePostMessage(cid:string){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async (input:MessageInput) => {
            const res = await apiClient.post<Message>(`/api/conversations/${cid}/messages`, input)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess : () => qc.invalidateQueries({ queryKey: ['messages', cid]})
    })
}