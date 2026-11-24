import {createFileRoute, Link} from '@tanstack/react-router'
import {useMutation} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import {useState} from "react";
import styles from "@/routes/form.module.css"
import type {ResetPasswordResponse} from "@/types.ts";
import {useCheckResetPassword} from "@/queries/auth.ts";

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
    const tokenCheck = useCheckResetPassword(token)
    const resetPasswordMutation = useMutation({
        mutationFn: async (new_password: string) => {
            const res = await apiClient.post<ResetPasswordResponse>('/api/auth/reset-password', {token, new_password})
            if (res.ok) {
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
    })

    function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault()
        resetPasswordMutation.mutate(newPassword)
    }

    const submitError = resetPasswordMutation.error
    const errDetails = submitError instanceof ApiRequestError ? submitError.apiError.details : undefined

    // optional: the link went bad between loading the page and submitting
    const linkDiedAtSubmit = submitError instanceof ApiRequestError
        && (submitError.httpStatus === 404 || submitError.httpStatus === 409)

    // optional: tell the user why the button is disabled
    const mismatch = confirmPassword !== "" && newPassword !== confirmPassword

    if (resetPasswordMutation.isSuccess) {
        return <div className={styles.form}>
            <h1>Reset Password</h1>
            <p role="status" className={styles.readOnlyLabel}>{resetPasswordMutation.data.message}</p>
            <Link className={'btn btn-primary'} to={'/login'}>Go to login</Link>
        </div>
    }
    if (!token || tokenCheck.isError || linkDiedAtSubmit) {
        return <div className={styles.form}>
            <h1>Reset Password</h1>
            <p role="alert" className={styles.error}>This reset link is invalid or has expired.</p>
            <p className={styles.readOnlyLabel}>Go to the login page and choose "Forgot password" to get a new one.</p>
            <Link className={'btn btn-primary'} to={'/login'}>Back to login</Link>
        </div>
    }
    if (tokenCheck.isPending) {
        return <div className={styles.form}>
            <p role="status">Checking your link…</p>
        </div>
    }

    return <form className={styles.form} onSubmit={handleSubmit}>
        <h1>Reset Password</h1>

        <label className={styles.label}>Password
            <input
                type="password"
                value={newPassword}
                onChange={e => setNewPassword(e.target.value)}
                aria-invalid={!!errDetails?.password}
                aria-describedby={errDetails?.password ? 'reset-password-error' : undefined}
            />
        </label>
        {errDetails?.password &&
            <p id="reset-password-error" className={styles.fieldError}>{errDetails.password}</p>}

        <label className={styles.label}>Confirm password
            <input
                type="password"
                value={confirmPassword}
                onChange={e => setConfirmPassword(e.target.value)}
                aria-invalid={mismatch}
                aria-describedby={mismatch ? 'reset-confirm-error' : undefined}
            />
        </label>
        {mismatch &&
            <p id="reset-confirm-error" className={styles.fieldError}>Passwords don't match.</p>}

        <button className="btn-primary" type="submit"
                disabled={resetPasswordMutation.isPending || newPassword.trim() === "" || newPassword !== confirmPassword}>
            Reset password
        </button>

        {submitError && <p role="alert" className={styles.error}>{submitError.message}</p>}
    </form>
}