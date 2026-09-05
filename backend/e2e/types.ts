//e2e/types.ts


export interface ApiEnvelope<T> {
    data: T
    correlation_id: string
    timestamp: string
}
export interface ApiErrorBody{
    error: {
        status: number
        code: string
        message: string
        correlation_id: string
        timestamp: string
    }
}

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

export interface UpdateInput {
    title?: string
    destination?: string
    meet_label?: string
    meet_lat?: number
    meet_lng?: number
    starts_at?:string
    max_size?: number
    host_seats?: number
    cost_per_seat_cents?: number
    difficulty?: 'easy' | 'moderate' | 'hard'
    pace?: 'relaxed' | 'moderate' | 'fast'
    notes?: string
}

export interface RegisterRequest{
    email: string
    password: string
    name: string
    experience: 'beginner'|'intermediate'|'experienced'
}

export interface MeResponse{
    email:string,
    name:string,
    experience:string,
    created_at: string,
    updated_at: string,
    verified_at: string|null,
}
export interface UpdateMeInput{
    name?: string,
    experience?: string
}

export interface MeRequest{
    name: string
    email: string
    experience: string
}
export interface MemberCard{
    id:string
    email?:string
    name:string
    experience:'beginner'|'intermediate'|'experienced'
}

export interface Hiker{
    id: string
    email: string
    name: string
    experience:'beginner'|'intermediate'|'experienced'
    home_area?: string
    bio?:string
    gender?:string
    created_at:string
    updated_at:string
    verified_at?:string
}

export interface VerifyEmailResponse{
    message: string
}

export const CODE_EMAIL_NOT_VERIFIED = "email_not_verified"