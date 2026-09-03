import {createFileRoute, Link } from '@tanstack/react-router'
import { useMutation} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import type {VerifyEmailResponse} from "@/types.ts";
import {useEffect} from "react";
import  styles  from "@/routes/form.module.css"


export const Route = createFileRoute('/verify-email')({
    validateSearch: (search: Record<string, unknown>) => {
        return {token: typeof search.token === 'string' ? search.token : ''}
    },
  component: VerifyEmailPage,
})

function VerifyEmailPage() {
    const {token} = Route.useSearch()
    const verifyEmailMutation = useMutation(
        {
            mutationFn: async (token:string) => {
              const res =
                  await apiClient.post<VerifyEmailResponse>('/api/auth/verify-email', {token})
                console.log('mutationFn got:', res)
              if (res.ok){
                  return res.data
              }
              throw new ApiRequestError(res.error, res.httpStatus)
            },
        }
    )

    useEffect(()=>{
        if (token){
            verifyEmailMutation.mutate(token)
        }
    }, [token])
    console.log('mutation status:', verifyEmailMutation.status, verifyEmailMutation.error)
    if (!token) return <div><p>Invalid verification token</p></div>
    if (verifyEmailMutation.isSuccess){
        return <div className={styles.form}>
            <p>{verifyEmailMutation.data?.message}</p>
            <p></p>
            <Link className={"btn btn-primary"} to={'/login'}>Log in</Link>
        </div>
    }
    if (verifyEmailMutation.isError){
        return <div className={styles.form}>
            <p className={styles.error}>{verifyEmailMutation.error.message}</p>
        </div>
    }
    return <div>loading..</div>

}
