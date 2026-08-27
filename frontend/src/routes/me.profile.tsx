import { createFileRoute } from '@tanstack/react-router'
import {requireAuth, useMeQuery} from "@/queries.ts";
import {ProfileForm} from "@/components/ProfileForm.tsx";

export const Route = createFileRoute('/me/profile')({
    beforeLoad : async ({context}) => requireAuth(context.queryClient),
    component: ProfilePage,
})

export function ProfilePage() {
    const { data: me, isPending, error } = useMeQuery()

    if (isPending) return <div>loading..</div>
    if (error) return <div>{error.message}</div>
    if (!me) return <div>no profile found</div>

    return <ProfileForm me={me} />

}
