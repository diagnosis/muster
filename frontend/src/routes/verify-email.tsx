import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { apiClient, ApiRequestError } from '@/lib/api.ts'
import type { VerifyEmailResponse } from '@/types.ts'

import styles from '@/routes/form.module.css'

export const Route = createFileRoute('/verify-email')({
  validateSearch: (search: Record<string, unknown>) => {
    return { token: typeof search.token === 'string' ? search.token : '' }
  },
  component: VerifyEmailPage,
})

function VerifyEmailPage() {
  const { token } = Route.useSearch()

  const verification = useQuery({
    queryKey: ['verify-email', token],
    queryFn: async () => {
      const res = await apiClient.post<VerifyEmailResponse>('/api/auth/verify-email', { token })
      if (res.ok) return res.data
      throw new ApiRequestError(res.error, res.httpStatus)
    },
    enabled: token !== '',
    retry: false,
    staleTime: Infinity,
    gcTime: Infinity,
    refetchOnWindowFocus: false,
  })

  if (!token)
    return (
      <div className={styles.form}>
        <p role={"alert"} className={styles.error}>This link is not valid.</p>
      </div>
    )
  if (verification.isSuccess) {
    return (
      <div className={styles.form}>
        <p role={"status"}>Your email is verified.</p>
        <Link className={'btn btn-primary'} to={'/login'}>
          Log in
        </Link>
      </div>
    )
  }
  if (verification.isError) {
    return (
      <div className={styles.form}>
        {verification.error instanceof ApiRequestError && verification.error.httpStatus === 409 ? (
          <p role={"alert"} className={styles.error}>This link has expired or was already used.</p>
        ) : (
          <p role={"alert"} className={styles.error}>This link is not valid.</p>
        )}
      </div>
    )
  }
  return <div>Verifying…</div>
}
