import {useQuery} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {PublicProfile} from "@/types.ts";


export function useHiker(id: string){
    return useQuery({
        queryKey:['hiker', id],
        queryFn: async () => {
            const res  = await apiClient.get<PublicProfile>(`/api/hikers/${id}`)
            if (res.ok){
                return  res.data
            }
            if (res.httpStatus === 404){
                return null
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        }
    })
}