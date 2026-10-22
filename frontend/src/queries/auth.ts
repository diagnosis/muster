// src/queries/auth.ts

import { useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {ForgotPasswordResponse, MeResponse, RequestLoginCodeResponse, ResendEmailResponse} from "@/types.ts";
import {forgetUserData} from "@/queries.ts";

export function useForgotPasswordMutation(){
 return   useMutation(
     {
         mutationFn: async (email:{email:string}) =>{
             const res = await apiClient.post<ForgotPasswordResponse>('/api/auth/forgot-password', email)
             if (res.ok){
                 return res.data
             }
             throw new ApiRequestError(res.error, res.httpStatus)
         }
     }
 )

}

export function useRequestLoginCodeMutation(){
    return useMutation({
        mutationFn: async(input:{email:string}) =>{
            const res = await apiClient.post<RequestLoginCodeResponse>('/api/auth/login-code', input)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        }
    })
}

export function useLoginWithCodeMutation() {
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async (input:{email:string, code:string})=>{
            const res = await apiClient.post<MeResponse>('/api/auth/login-code/verify', input)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            forgetUserData(qc)
            qc.invalidateQueries({queryKey: ['me']})
        }
    })
}

export function useLoginWithPasswordMutation() {
    const qc = useQueryClient()
    return useMutation({
        mutationFn: async(creds : {email: string, password: string}) => {
            const res = await apiClient.post<MeResponse>('/api/auth/login', creds)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess: () => {
            forgetUserData(qc)
            qc.invalidateQueries({queryKey: ['me']})
        }
    })
}

export function useResendMutation(){
    return useMutation({
        mutationFn: async (email:{email:string}) => {
            const res = await apiClient.post<ResendEmailResponse>('/api/auth/resend-verification', email)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        }
    })
}
