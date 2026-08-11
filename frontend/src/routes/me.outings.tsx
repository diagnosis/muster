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
    return <div className={styles.page}>
        <section className={styles.section}>
            <h2 className={styles.heading}>Hosting</h2>
            {hostingOutings.length === 0
                ? <div className={styles.empty}>
                    <p>Not hosting anything yet.</p>
                    <Link className="btn-primary btn" to="/outings/new">Create an outing</Link>
                </div>
                : <div className={styles.list}>
                    {hostingOutings.map(o => (
                        <Link className={styles.cardLink} key={o.id} to="/outings/$id" params={{id: o.id}}>
                            <OutingCard outing={o}/>
                        </Link>
                    ))}
                </div>
            }
        </section>

        <section className={styles.section}>
            <h2 className={styles.heading}>Joined</h2>
            {joinedOutings.length === 0
                ? <p className={styles.emptyText}>Not joined anything yet.</p>
                : <div className={styles.list}>
                    {joinedOutings.map(o => (
                        <Link className={styles.cardLink} key={o.id} to="/outings/$id" params={{id: o.id}}>
                            <OutingCard outing={o}/>
                        </Link>
                    ))}
                </div>
            }
        </section>
    </div>

}
