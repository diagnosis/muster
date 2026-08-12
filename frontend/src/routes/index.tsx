// src/routes/index.tsx
import {createFileRoute, Link} from '@tanstack/react-router'
import {OutingCard} from "../components/OutingCard.tsx";
import styles from "./index.module.css"
import {useOutings} from "../queries.ts";

export const Route = createFileRoute('/')({
  component: OutingsPage,
})

function OutingsPage() {

    //using tanstack query
    const {data: outings, isPending, error} = useOutings()
    if (isPending){
        return <div>Loading outings...</div>
    }

    if (error){
        return <div>
            error loading outings: {error.message}
        </div>
    }
    return <>
        <h2 className={styles.heading}>Outings</h2>
        <div className={styles.list}>

            {outings.map(outing=>
                <Link className={styles.cardLink} to={"/outings/$id"} params={{id: outing.id}} key={outing.id}><OutingCard outing={outing}/></Link>

            )}
        </div>
    </>
}
