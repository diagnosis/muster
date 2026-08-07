import {createFileRoute, Link, redirect} from '@tanstack/react-router'
import {meQueryOptions, useMyOutings} from "../queries.ts";
import {OutingCard} from "../components/OutingCard.tsx";



export const Route = createFileRoute('/me/outings')({
    beforeLoad: async ({context}) => {
       const me = await context.queryClient.ensureQueryData(meQueryOptions())
        if (!me) throw redirect({to:'/login'})
    },
  component: MeOutingsPage,
})

function MeOutingsPage() {
    const {data, isPending, error} = useMyOutings()
    if (isPending) return <div>Outings loading...</div>
    if (error) return <div>{error.message}</div>
    const hostingOutings = data.hosting
    const joinedOutings = data.joined
    return (
        <div>
            <h2>Hosting</h2>
    {hostingOutings.length === 0
    ? <p>Not hosting anything yet.</p>
    : hostingOutings.map(o=> (
        <Link key={o.id} to={"/outings/$id"} params={{id:o.id}}>
            <OutingCard outing={o}/>
        </Link>
        ))}

           <h2>Joined</h2>
            {joinedOutings.length === 0
            ? <p>Not joined anything yet.</p>
                : joinedOutings.map(o => (
                       <Link key={o.id} to={"/outings/$id"} params={{id:o.id}}>
                           <OutingCard outing={o}/>
                       </Link>
                ))
            }
        </div>
    )

}
