// src/utils/parser.ts

export function parsePositiveInt(s:string): number | null{
    const n = Number(s)
    if (Number.isInteger(n) && n>0){
        return n
    }
    return null
}

export function parseNonNegativeInt(s:string): number | null {
    const n = Number(s)
    if (Number.isInteger(n) && n >= 0){
        return n
    }
    return null
}
export function parseNonNegativeFloat(s: string, decimal: number): number | null {
    const trimmed = s.trim();
    if (trimmed === "") return null;

    const num = Number(trimmed);
    if (Number.isNaN(num) || num < 0) return null;

    const dotIndex = trimmed.indexOf('.');
    if (dotIndex !== -1) {
        const actualDecimals = trimmed.length - dotIndex - 1;
        if (actualDecimals > decimal) return null;
    }

    return num;
}
