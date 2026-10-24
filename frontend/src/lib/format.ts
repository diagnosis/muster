import type {Capacity, CapacityTone} from "@/types.ts";


export function listNames(names: string[]): string {
    if (names.length <= 1) return names[0] ?? ''
    return `${names.slice(0, -1).join(', ')} and ${names[names.length - 1]}`
}

export function ordinal(n: number): string {
    const s = ['th', 'st', 'nd', 'rd'], v = n % 100
    return n + (s[(v - 20) % 10] ?? s[v] ?? s[0])
}

export const initial = (name: string) => name.charAt(0).toUpperCase()

export const plural = (count: number, text: string) =>
    `${count} ${text}${count === 1 ? '' : 's'}`

export function capacityLine(c: Capacity, maxSize: number): {text: string, tone: CapacityTone} {
    const head = `${c.people_count} of ${maxSize} going`

    let clause: string
    let tone: CapacityTone
    if (c.seats_short > 0) {
        clause = `${plural(c.seats_short, 'seat')} short, a driver would help`
        tone = 'short'
    } else if (c.spots_left === 0 && c.people_count < maxSize) {
        clause = 'no seats left, a driver would open more'
        tone = 'short'
    } else if (c.spots_left === 0) {
        clause = 'full'
        tone = 'full'
    } else {
        clause = `${plural(c.spots_left, 'spot')} left`
        tone = 'open'
    }

    const waiting = c.waiting_count > 0 ? `${c.waiting_count} waiting` : ''
    const text = [head, clause, waiting].filter(Boolean).join(' · ')
    return {text, tone}
}