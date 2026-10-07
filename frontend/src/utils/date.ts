//src/utils/date.ts

export function isoToLocalInput(iso: string): string {
    const d = new Date(iso)
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
    return d.toISOString().slice(0, 16)
}

export function relativeTime(createdAt: string): string {
    const then = new Date(createdAt).getTime()
    const diffSec = Math.round((Date.now() - then) / 1000)

    if (diffSec < 60)     return 'just now'
    const diffMin = Math.round(diffSec / 60)
    if (diffMin < 60)     return `${diffMin}m ago`
    const diffHr = Math.round(diffMin / 60)
    if (diffHr < 24)      return `${diffHr}h ago`
    const diffDay = Math.round(diffHr / 24)
    if (diffDay < 7)      return `${diffDay}d ago`
    // older than a week: show the date
    return new Date(createdAt).toLocaleDateString()
}

// "4:12 PM" for today, "Oct 2, 4:12 PM" for older messages.
export function messageTime(iso: string): string {
    const d = new Date(iso)
    const time = d.toLocaleTimeString([], {hour: 'numeric', minute: '2-digit'})
    if (d.toDateString() === new Date().toDateString()) return time
    return `${d.toLocaleDateString([], {month: 'short', day: 'numeric'})}, ${time}`
}

// "Sat, Oct 10 · 6:00 AM"
export function outingWhen(iso: string): string {
    const d = new Date(iso)
    const date = d.toLocaleDateString('en-US', {weekday: 'short', month: 'short', day: 'numeric'})
    const time = d.toLocaleTimeString('en-US', {hour: 'numeric', minute: '2-digit'})
    return `${date} · ${time}`
}

// "Sat, Oct 10 · 6:00 AM"                           no end time
// "Sat, Oct 10 · 6:00 AM – 6:00 PM"                 ends the same day
// "Sat, Oct 10 · 6:00 AM – Sun, Oct 11 · 4:00 PM"   ends another day
export function outingRange(startIso: string, endIso?: string | null): string {
    const start = outingWhen(startIso)
    if (!endIso) return start
    const s = new Date(startIso)
    const e = new Date(endIso)
    if (s.toDateString() === e.toDateString()) {
        return `${start} – ${e.toLocaleTimeString('en-US', {hour: 'numeric', minute: '2-digit'})}`
    }
    return `${start} – ${outingWhen(endIso)}`
}