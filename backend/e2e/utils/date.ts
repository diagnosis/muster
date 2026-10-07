


// an ISO timestamp N days from now, at the given hour
export function at(daysFromNow: number, hour: number): string {
    const d = new Date()
    d.setDate(d.getDate() + daysFromNow)
    d.setHours(hour, 0, 0, 0)
    return d.toISOString()
}

// an ISO timestamp exactly `hours` after another one
export function plusHours(iso: string, hours: number): string {
    return new Date(new Date(iso).getTime() + hours * 3600 * 1000).toISOString()
}
export const sameTime = (a: string, b: string) => new Date(a).getTime() === new Date(b).getTime()