// src/queries.ts


import type {MeResponse} from "./types.ts";
import {apiClient} from "./lib/api.ts";
import {useQuery} from "@tanstack/react-query";


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

export function useMeQuery() {
    return useQuery(
        {
            queryKey: ['me'],
            queryFn: getMe,
            staleTime: 5 * 60 * 1000,
        }
    )
}