// src/routes/index.tsx
import {createFileRoute, Link} from '@tanstack/react-router'
import {OutingCard} from "@/components/OutingCard.tsx";
import styles from "@/routes/index.module.css"
import {useOutings} from "@/queries/outing.ts";
import {useEffect} from "react";
import {useEvents} from "@/events/EventProvider.tsx";
import {useQueryClient} from "@tanstack/react-query";



export const Route = createFileRoute('/')({
  component: OutingsPage,
})

export function OutingsPage() {

    //using tanstack query
    const {data: outings, isPending, error} = useOutings()
    const qc = useQueryClient()
    const { subscribe } = useEvents()
    useEffect(()=> {
        return subscribe('outing.changed', ()=>{
                qc.invalidateQueries({ queryKey: ['outings'] })
        })
    }, [subscribe, qc])

    if (isPending){
        return <div>Loading outings...</div>
    }

    if (error){
        return <div>
            error loading outings: {error.message}
        </div>
    }


    return <>
        <h1 className={styles.heading}>Outings</h1>


            {outings.length === 0
                ? <div className={styles.empty}>
                    <p className={styles.emptyText}>No outings yet.</p>
                    <Link className={'btn-primary btn'} to={'/outings/new'}>Create outing</Link>
                </div>
               : <div className={styles.list}>
                    {outings.map(outing=>
                    <Link className={styles.cardLink} to={"/outings/$id"} params={{id: outing.id}} key={outing.id}><OutingCard outing={outing}/></Link>)}
                </div>

            }


    </>
}
