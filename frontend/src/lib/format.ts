

export function listNames(names: string[]): string {
    if (names.length <= 1) return names[0] ?? ''
    return `${names.slice(0, -1).join(', ')} and ${names[names.length - 1]}`
}

export function ordinal(n: number): string {
    const s = ['th', 'st', 'nd', 'rd'], v = n % 100
    return n + (s[(v - 20) % 10] ?? s[v] ?? s[0])
}

export const initial = (name: string) => name.charAt(0).toUpperCase()