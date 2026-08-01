// frontend/lib.api.ts

import type {ApiResponse} from '../types'


export const apiClient = {
    get : async <T>(endpoint:string):Promise<ApiResponse<T>> => {
        const res = await fetch(endpoint)
        const json = await res.json().catch(()=> null)
        if (res.ok){
            return {ok: true, data: json?.data as T}
        }
        return {
            ok: false,
            httpStatus: res.status,
            error: json?.error ?? {status:0, message: res.statusText || 'Request failed', timestamp: new Date().toISOString()}
        }

    }
}