// src/queries/comment.ts

import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {Comment, CommentInput, ListCommentResponse} from "@/types.ts";
import {queryOptions, useMutation, useQuery, useQueryClient} from "@tanstack/react-query";



export async function listComments(outingID:string){
    const res = await apiClient.get<ListCommentResponse>(`/api/outings/${outingID}/comments`)
    if (res.ok){
        return res.data
    }
    throw new ApiRequestError(res.error, res.httpStatus)

}

export const commentsQueryOption = (outingID:string) => (
    queryOptions(
        {
            queryKey:['comments', outingID],
            queryFn: () => listComments(outingID),
            refetchInterval: 30_000
        }
    )
)

export function useListComments(outingID:string){
    return (
        useQuery(commentsQueryOption(outingID))
    )
}

export function useAddComment(outingID:string) {
    const qc = useQueryClient()
    return useMutation(
        {
            mutationFn: async (input:CommentInput) =>{
                const res = await apiClient.post<Comment>(`/api/outings/${outingID}/comments`, input)
                if (res.ok){
                    return res.data
                }
                throw new ApiRequestError(res.error, res.httpStatus)
            },
            onSuccess: () => qc.invalidateQueries({queryKey:['comments', outingID]})
        }
    )
}
export function useDeleteComment(outingID:string){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async (commentID:string) => {
            const res = await apiClient.del(`/api/outings/${outingID}/comments/${commentID}`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => qc.invalidateQueries({queryKey:['comments', outingID]})
    })
}
export function useLikeComment(outingID:string){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async (commentID:string) => {
            const res = await apiClient.post(`/api/outings/${outingID}/comments/${commentID}/like`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => qc.invalidateQueries({queryKey:['comments', outingID]})

    })
}
export function useUnlikeComment(outingID:string){
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async (commentID:string) => {
            const res = await apiClient.del(`/api/outings/${outingID}/comments/${commentID}/like`)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => qc.invalidateQueries({queryKey:['comments', outingID]})
    })
}