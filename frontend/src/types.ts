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
    code: string
    message: string
    details?: Record<string,string>
    correlation_id?: string
    timestamp: string
}

export type Difficulty = 'easy' | 'moderate' | 'hard'
export type Pace = 'relaxed' | 'moderate' | 'fast'
export type OutingStatus = 'open' | 'cancelled'
export type Phase = 'upcoming' | 'in_progress' | 'past'
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
    conversation_id:string
    ends_at: string | null
    phase: Phase
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
    request_id?: string
    hiker_id: string
    name: string
    experience: Experience
}

export type RequestStatus = 'requested'|'accepted'|'declined'|'withdrawn'|'removed'
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
export interface JoinRequestInput {
    role: HikerRole
    seats_offered: number
    guests: number
    note?: string
}

export interface MeResponse{
    id:string
    email:string,
    name:string,
    experience:Experience,
    created_at: string,
    updated_at: string,
}

export interface MeInputRequest{
    name?:string
    experience?:Experience
}

export interface RegisterRequest{
    email: string
    password: string
    name: string
    experience: Experience
}

export interface PendingRequestResponse extends JoinRequest{
    hiker_name: string
    hiker_experience: Experience
}

export interface MyOutings{
    hosting: Outing[]
    joined: Outing[]
}

export interface CreateOutingInput {
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
    ends_at?: string
    clear_ends_at?: boolean
}

export interface VerifyEmailResponse {
    message: string
}

export interface ResendEmailResponse{
    message: string
}

export interface ForgotPasswordResponse{
    message: string
}

export interface ResetPasswordResponse{
    message: string
}

export interface NotificationsResponse{
    notifications:NotificationEvent[]
    unread_count:number
}
export interface NotificationEvent{
    id: string
    hiker_id: string
    kind: string
    payload: {outing_id:string,outing_title:string, from_name:string, conversation_id:string}
    created_at: string
    read_at:  string|null
}

export interface CommentView{
    id: string
    outing_id: string
    hiker_id: string
    parent_id: string|null
    body: string
    created_at: string
    deleted: boolean
    author_name:string
    like_count: number
    liked_by_me:boolean
}
export interface ListCommentResponse{
    comments:CommentView[]
}

export interface CommentInput{
    body:string
    parent_id:string|null
}
export interface Comment{
    id: string
    outing_id: string
    hiker_id: string
    parent_id: string|null
    body: string
    created_at: string
}

export interface ListMessagesResponse{
    messages: Message[]
}
export interface MessageInput{
    body: string
}

export interface Message {
    id: string
    conversation_id: string
    hiker_id: string
    seq: number
    body: string
    created_at: string
}

export interface Conversation{
    id: string
    kind: ConversationKind
    outing_id: string | null
    dm_a: string | null
    dm_b: string | null
    dm_initiator: string | null
    dm_status: DMStatus | null
    dm_declined_by: string | null
    created_at: string
}
export interface Participant{
    hiker_id: string
    name: string
}
export interface ConversationView{
    id: string
    kind: ConversationKind
    outing_title: string | null
    outing_starts_at: string | null
    participants: Participant[]
    outing_id: string | null
    outing_host_id: string | null
    dm_a: string | null
    dm_b: string | null
    dm_initiator: string | null
    dm_status: DMStatus | null
    dm_declined_by: string | null
    created_at: string
}


export interface ListConversationsResponse{
    conversations: ConversationSummary[]
}
export type ConversationKind = "outing" | "dm"
export type DMStatus = "pending" | "accepted" | "declined"
export interface ConversationSummary{
    id: string
    kind: ConversationKind
    title: string,
    dm_status: DMStatus | null
    dm_initiator: string | null
    dm_declined_by: string | null
    last_message_at: string | null
    last_preview: string
    created_at: string
    unread_count: number
}


