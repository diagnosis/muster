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
