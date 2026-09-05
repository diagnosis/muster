import {createFileRoute, Link} from '@tanstack/react-router'
import {useMutation} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import {useState} from "react";
import styles from "@/routes/form.module.css"
import type {ResetPasswordResponse} from "@/types.ts";

export const Route = createFileRoute('/reset-password')({
    validateSearch: (search: Record<string, unknown>) => {
        return {token: typeof search.token === 'string' ? search.token : ''}
    },
    component: ResetPasswordPage,
})

export function ResetPasswordPage() {
    const {token} = Route.useSearch()
    const [newPassword, setNewPassword] = useState("")
    const [confirmPassword, setConfirmPassword] = useState("")
    const resetPasswordMutation = useMutation({
        mutationFn: async (new_password:string)=>{
            const res = await apiClient.post<ResetPasswordResponse>('/api/auth/reset-password', {token, new_password})
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
    })

    function handleSubmit(e:React.SubmitEvent){
        e.preventDefault()
        resetPasswordMutation.mutate(newPassword)
    }
    if (!token){
        return <div className={styles.form}>
            <p className={styles.error}>Invalid Token</p>
        </div>
    }
    if (resetPasswordMutation.isSuccess){
        return <>
            {resetPasswordMutation.isSuccess&&
                <div className={styles.form}>
                    <h1>Reset Password</h1>
                    <p className={styles.readOnlyLabel}>{resetPasswordMutation.data.message}</p>
                    <Link className={'btn btn-primary'} to={'/login'}>Go to login</Link>
                </div>
            }
        </>
    }
  return <form className={styles.form}
               onSubmit={handleSubmit}>
      <h1>Reset Password</h1>
      <label className={styles.label}>Password
          <input
              type="password"
              value={newPassword}
              onChange={e => setNewPassword(e.target.value)}
          />
      </label>
      <label className={styles.label}>Confirm password
          <input
              type="password"
              value={confirmPassword}
              onChange={e => setConfirmPassword(e.target.value)}
          />
      </label>

      <button className="btn-primary"
              type="submit" disabled={resetPasswordMutation.isPending || newPassword.trim() == "" || newPassword !== confirmPassword}>Reset password</button>
      {resetPasswordMutation.error&&<p className={styles.error}>{resetPasswordMutation.error.message}</p>}

  </form>
}
