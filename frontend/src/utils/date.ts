//src/utils/date.ts

export function isoToLocalInput(iso: string): string {
    const d = new Date(iso)
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
    return d.toISOString().slice(0, 16)
}