// frontend/lib.api.ts

import type {ApiResponse} from '../types'



const request = async <T>(endpoint:string, init?: RequestInit): Promise<ApiResponse<T>> =>{
    const res = await fetch(endpoint, init)
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
}