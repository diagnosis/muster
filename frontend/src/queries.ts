// src/queries.ts


import type {Detail, MeResponse, MyOutings, Outing, PendingRequestResponse} from "./types.ts";
import {apiClient} from "./lib/api.ts";
import {type QueryClient, useQuery} from "@tanstack/react-query";
import {redirect} from "@tanstack/react-router";


export const  getMe = async () => {
 const res =  await apiClient.get<MeResponse>("/api/auth/me")
    if (res.ok){
        return res.data
    }else{
        if (res.httpStatus === 401){
            return null
        }else{
            throw new Error(res.error.message)
        }
    }
}

export const meQueryOptions = () =>({
    queryKey: ['me'],
    queryFn: getMe,
    staleTime: 5 * 60 * 1000,
})

export function useMeQuery() {
    return useQuery(
        meQueryOptions()
    )
}


export const getOutingByID = async (id:string) => {
    const res = await apiClient.get<Detail>(`/api/outings/${id}`)
    if (res.ok){
        return res.data
    }
    throw new Error(res.error.message)
}
export function useOuting(id:string){
    return useQuery({
        queryKey:['outing', id],
        queryFn: () => getOutingByID(id),
    })
}

export const getOutings=async ()=>{
    const res = await apiClient.get<Outing[]>('/api/outings')
    if (res.ok){
        return res.data
    }
    throw new Error(res.error.message)
}

export function useOutings(){
    return useQuery({
        queryKey:['outings'],
        queryFn:getOutings,
    })
}



export function useOutingJoinRequests(id: string) {
    return useQuery(
        {
            queryKey:['outing-join-requests', id],
            queryFn: async () => {
                const res = await apiClient.get<PendingRequestResponse[]>(`/api/outings/${id}/requests`)
                if(res.ok){
                    return res.data
                }
                throw new Error(res.error.message)
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
                throw new Error(res.error.message)
            }
        }
    )
}

export async function requireAuth(queryClient: QueryClient){
    const me = await queryClient.ensureQueryData(meQueryOptions())
    if (!me) throw redirect({to:'/login'})
}