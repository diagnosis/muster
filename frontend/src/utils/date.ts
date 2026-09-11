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