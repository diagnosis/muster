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

