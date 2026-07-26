//e2e/types.ts

export interface OutingResponse {
    id: string
    host_id: string
    title: string
    destination: string
    meet_label: string
    starts_at: string
    max_size: number
    host_seats: number
    cost_per_seat_cents: number
    difficulty: 'hard'|'easy'|'moderate'
    pace: 'relaxed' | 'fast' | 'moderate'
}

export interface JoinRequestResponse {
    id: string
    outing_id: string
    hiker_id: string
    status: 'requested'|'accepted'|'declined'|'withdrawn'
    role: 'rider'|'driver'
    seats_offered: number
    guests: number
}

export interface MemberResponse {
    hiker_id: string
    name: string
    experience: string
}

export interface DetailResponse {
    outing: OutingResponse
    host: MemberResponse
    roster: MemberResponse[]
    my_request?: JoinRequestResponse      // optional — absent for anonymous (omitempty)
    seat_capacity: number
    people_count: number
    seats_short: number
    spots_left: number
}