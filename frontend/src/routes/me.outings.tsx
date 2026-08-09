import {createFileRoute, Link} from '@tanstack/react-router'
import {requireAuth, useMyOutings} from "../queries.ts";
import {OutingCard} from "../components/OutingCard.tsx";
import styles from "./me.outings.module.css"



export const Route = createFileRoute('/me/outings')({
    beforeLoad: async ({context}) => requireAuth(context.queryClient),
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
            <h2 className={styles.heading}>Hosting</h2>
    {hostingOutings.length === 0
    ? <p>Not hosting anything yet.</p>
    : hostingOutings.map(o=> (
        <Link key={o.id} to={"/outings/$id"} params={{id:o.id}}>
            <OutingCard outing={o}/>
        </Link>
        ))}

           <h2 className={styles.heading}>Joined</h2>
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
