// src/queries.ts


import type { MeResponse, NotificationsResponse } from "@/types.ts";
import {apiClient, ApiRequestError, refreshSession} from "@/lib/api.ts";
import {type InvalidateQueryFilters, type QueryClient, useQuery} from "@tanstack/react-query";
import {redirect} from "@tanstack/react-router";


// invalidateQueries, but safe during a query's first load.
// TanStack reuses an in-flight first fetch instead of restarting it, and that
// fetch may predate the change. If one is in flight, wait for it, then go again.
export async function invalidateFresh(qc: QueryClient, filters?: InvalidateQueryFilters) {
    const firstLoad = qc.getQueryCache().findAll(filters)
        .some(q => q.state.fetchStatus === 'fetching' && q.state.data === undefined)
    await qc.invalidateQueries(filters)
    if (firstLoad) await qc.invalidateQueries(filters)
}


export const getMe = async () => {
    let res = await apiClient.probe<MeResponse>("/api/auth/me")
    if (!res.ok && res.httpStatus === 401 && await refreshSession() === 200) {
        res = await apiClient.probe<MeResponse>("/api/auth/me")
    }
    if (res.ok) return res.data
    if (res.httpStatus === 401) return null
    throw new ApiRequestError(res.error, res.httpStatus)
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



export async function requireAuth(queryClient: QueryClient){
    const me = await queryClient.ensureQueryData(meQueryOptions())
    if (!me) throw redirect({to:'/login'})
}

export async function requireGuest(queryClient: QueryClient){
    const me = await queryClient.ensureQueryData(meQueryOptions())
    if (me) throw redirect({to:'/'})
}

export function forgetUserData(qc: QueryClient) {
    qc.removeQueries({ predicate: q => q.queryKey[0] !== 'me' })
}

export function useNotifications() {
    return useQuery({
        queryKey: ['notifications'],
        queryFn: async () => {
            const res = await apiClient.get<NotificationsResponse>('/api/notifications')
            if (res.ok) return res.data
            throw new ApiRequestError(res.error, res.httpStatus)
        },
    })
}


