

export type Experience = 'beginner'|'intermediate'|'experienced'

export interface RegisterRequest{
    name: string
    email: string
    password: string
    experience: Experience
}