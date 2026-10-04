// src/components/Icons.tsx
// Small stroke icons. They take the text colour of whatever they sit in.

import type {ReactNode} from "react";

interface IconProps {
    size?: number
}

function Svg({size = 20, children}: IconProps & {children: ReactNode}) {
    return (
        <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
             strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            {children}
        </svg>
    )
}

export function BellIcon({size = 22}: IconProps) {
    return <Svg size={size}><path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/></Svg>
}

export function TrashIcon({size = 18}: IconProps) {
    return <Svg size={size}><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></Svg>
}

export function ChevronLeftIcon({size = 16}: IconProps) {
    return <Svg size={size}><path d="M15 18l-6-6 6-6"/></Svg>
}

export function MountainIcon({size = 20}: IconProps) {
    return <Svg size={size}><path d="M8 3l4 8 5-5 5 15H2L8 3z"/></Svg>
}

export function MenuIcon({size = 22}: IconProps) {
    return <Svg size={size}><path d="M3 6h18"/><path d="M3 12h18"/><path d="M3 18h18"/></Svg>
}

export function CalendarIcon({size = 18}: IconProps) {
    return <Svg size={size}><path d="M3 9h18"/><path d="M8 3v4"/><path d="M16 3v4"/><path d="M5 5h14a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2z"/></Svg>
}

export function PinIcon({size = 18}: IconProps) {
    return <Svg size={size}><path d="M12 21s-7-5.6-7-11a7 7 0 0 1 14 0c0 5.4-7 11-7 11z"/><path d="M12 12.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z"/></Svg>
}

export function FlagIcon({size = 18}: IconProps) {
    return <Svg size={size}><path d="M5 21V4"/><path d="M5 4h12l-2.5 4L17 12H5"/></Svg>
}

export function MessageIcon({size = 20}: IconProps) {
    return <Svg size={size}><path d="M4 5h16v11H9l-5 4V5z"/></Svg>
}

// filled when liked, outline otherwise
export function HeartIcon({size = 16, filled = false}: IconProps & {filled?: boolean}) {
    return (
        <svg width={size} height={size} viewBox="0 0 24 24" fill={filled ? "currentColor" : "none"}
             stroke="currentColor" strokeWidth={1.8} strokeLinejoin="round" aria-hidden="true">
            <path d="M12 20s-7-4.4-7-9.5A4 4 0 0 1 12 8a4 4 0 0 1 7 2.5C19 15.6 12 20 12 20z"/>
        </svg>
    )
}
