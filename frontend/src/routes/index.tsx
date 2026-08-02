// src/routes/index.tsx
import {createFileRoute, Link} from '@tanstack/react-router'
import {apiClient} from "../lib/api.ts";
import type {Outing} from "../types.ts";
import {useQuery} from "@tanstack/react-query";
import {OutingCard} from "../components/OutingCard.tsx";
import '../index.css'

export const Route = createFileRoute('/')({
  component: OutingsPage,
})

function OutingsPage() {
    async function getOutings(){
        const res = await apiClient.get<Outing[]>('/api/outings')
        if (res.ok){
            return res.data
        }
        throw new Error(res.error.message)
    }
    //using tanstack query
    const {data: outings, isPending, error} = useQuery(
        {
            queryKey:['outings'],
            queryFn: getOutings
        }
    )
    if (isPending){
        return <div>Loading outings...</div>
    }

    if (error){
        return <div>
            error loading outings: {error.message}
        </div>
    }
    return <>
        <div className='wrapper'>
            <h2>Outings</h2>
            {outings.map(outing=>
                <Link to={"/outings/$id"} params={{id: outing.id}} key={outing.id}><OutingCard outing={outing}/></Link>

            )}
        </div>
    </>
}
