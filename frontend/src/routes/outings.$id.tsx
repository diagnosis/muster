import { createFileRoute } from '@tanstack/react-router'
import {apiClient} from "../lib/api.ts";
import type {Detail} from "../types.ts";
import {useQuery} from "@tanstack/react-query";

export const Route = createFileRoute('/outings/$id')({
  component: OutingDetailPage,
})

function OutingDetailPage() {
    const {id} = Route.useParams()
    async function getOutingByID(id:string){
       const res = await apiClient.get<Detail>(`/api/outings/${id}`)
        if (res.ok){
            return res.data
        }
        throw new Error(res.error.message)
    }

    const {data:detail, isPending, error} = useQuery(
        {
            queryKey : ['outing', id],
            queryFn:() => getOutingByID (id)
        }
    )
    // Handle loading and error states before rendering
    if (isPending) return <div>Loading...</div>
    if (error) return <div>Error: {error.message}</div>
    const date = new Date(detail.outing.starts_at)
    const starts_at_date =
        date.toLocaleDateString('en-US', {
            weekday: 'short',
            month:'short',
            day:'numeric'
        })
    const starts_at_time=
        date.toLocaleTimeString('en-US', {
            hour:'numeric',
            minute:'2-digit'
        })

    const effectiveCap = Math.min(detail.seat_capacity, detail.outing.max_size)
    const isFull = detail.people_count >= detail.outing.max_size

    return <>
        <div>
            <h1>{detail.outing.title}</h1>
            <p>{detail.outing.destination} · {starts_at_date} · {starts_at_time}</p>
            <a href="#">Join Button</a>
            <p>{detail.people_count} going · {detail.spots_left} of {effectiveCap} spots left</p>
            {isFull && <p>This outing is full.</p>}
            {!isFull && detail.seats_short > 0 && <p>⚠️ {detail.seats_short} more seats needed — join as a driver?</p>}
            {!isFull && detail.seats_short === 0 && detail.spots_left === 0 && <p>No seats left — a driver could open more spots. 🚗</p>}
            <h3>Host</h3>
            <p>{detail.host.name} · {detail.host.experience}</p>

            <h3>Who's going ({detail.roster.length})</h3>
            {detail.roster.length === 0
                ? <p>No members yet — be the first.</p>
                : detail.roster.map(m => <p key={m.hiker_id}>{m.name} · {m.experience}</p>)}

            {detail.outing.notes && <p>{detail.outing.notes}</p>}
        </div>
    </>
}
