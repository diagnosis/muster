//frontend/src/types.ts



export type ApiResponse<T> = {
    ok: true;
    data: T;
}|{
    ok: false
    httpStatus: number
    error: ApiError

}

export interface ApiError{
    status: number
    message: string
    details?: Record<string,string>
    correlation_id?: string
    timestamp: string
}

export type Difficulty = 'easy' | 'moderate' | 'hard'
export type Pace = 'relaxed' | 'moderate' | 'fast'
export type OutingStatus = 'open' | 'cancelled'
export interface Outing{
    id: string
    host_id: string
    title: string
    destination: string
    meet_label: string
    meet_lat?: number
    meet_lng?: number
    starts_at: string
    max_size: number
    host_seats: number
    cost_per_seat_cents: number
    difficulty: Difficulty
    pace: Pace
    notes?: string
    status: OutingStatus
    created_at: string
    updated_at: string
}

export interface Detail{
    outing: Outing
    host: Member
    roster: Member[]
    my_request?: JoinRequest
    seat_capacity: number
    people_count: number
    seats_short: number
    spots_left: number
}

export type Experience = 'beginner'|'intermediate'|'experienced'
export interface Member{
    hiker_id: string
    name: string
    experience: Experience
}

export type RequestStatus = 'requested'|'accepted'|'declined'|'withdrawn'
export type HikerRole = 'rider'|'driver'
export interface JoinRequest{
    id: string
    outing_id: string
    hiker_id: string
    status: RequestStatus
    role: HikerRole
    seats_offered: number
    guests: number
    note?: string
    created_at: string
    updated_at: string

}
