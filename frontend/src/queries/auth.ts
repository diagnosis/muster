// src/queries/auth.ts

import {useMutation} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {ForgotPasswordResponse} from "@/types.ts";

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