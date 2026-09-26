// frontend/lib.api.ts
const API_BASE = import.meta.env.VITE_API_URL ?? ''
import type {ApiError, ApiResponse} from '../types'


let refreshInFlight: Promise<number> | null = null

export const refreshSession = (): Promise<number> => {
    if (refreshInFlight) return refreshInFlight
    refreshInFlight = bareRequest('/api/auth/refresh', { method: 'POST' })
        .then(r => (r.ok ? 200 : r.httpStatus))
        .catch(() => 0)                       // fetch threw: server unreachable
        .finally(() => { refreshInFlight = null })
    return refreshInFlight
}

const request = async <T>(endpoint:string, init?: RequestInit):Promise<ApiResponse<T>> => {
    const res = await bareRequest<T>(endpoint, init)
    if (res.ok){
        return res
    }
    if(res.httpStatus !== 401){
        return res
    }
    const refresh = await refreshSession()
    if(refresh === 200){
        return bareRequest<T>(endpoint, init)
    }
    return res
}




const bareRequest = async <T>(endpoint:string, init?: RequestInit): Promise<ApiResponse<T>> =>{
    const res = await fetch(API_BASE+endpoint, {...init, credentials: 'include'})
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


export const apiClient = {
    get :  <T>(endpoint:string) =>  request<T>(endpoint),
    post:  <T>(endpoint:string, body?:unknown) =>
       request<T>(endpoint, {
           method:'POST',
           headers: {'Content-Type': 'application/json'},
           body: JSON.stringify(body),
       }),
    del:  <T>(endpoint:string): Promise<ApiResponse<T>> => request<T>(endpoint, {method:'DELETE'}),
    patch: <T>(endpoint:string, body?:unknown) => request<T>(endpoint, {
        method: 'PATCH',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(body),
    })
}


export class ApiRequestError extends Error {
    readonly apiError: ApiError
    readonly httpStatus: number
    constructor(apiError: ApiError, httpStatus: number) {
        super(apiError.message)
        this.apiError = apiError
        this.httpStatus = httpStatus
    }
}