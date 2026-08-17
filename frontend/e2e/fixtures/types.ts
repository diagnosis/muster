


export type Difficulty = 'easy' | 'moderate' | 'hard'
export type Pace = 'relaxed' | 'moderate' | 'fast'
export type Experience = 'beginner'|'intermediate'|'experienced'

export interface RegisterRequest{
    name: string
    email: string
    password: string
    experience: Experience
}

export interface CreateOutingInput{
    title: string
    destination: string
    meet_label: string
    starts_at: string
    max_size: number
    host_seats: number
    cost_per_seat_cents: number
    difficulty: Difficulty
    pace: Pace
    notes?: string
}

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

export type HikerRole = 'rider'|'driver'
export interface JoinRequestInput {
    role: HikerRole
    seats_offered: number
    guests: number
    note?: string
}
export type RequestStatus = 'requested'|'accepted'|'declined'|'withdrawn'
export interface JoinRequest {
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
