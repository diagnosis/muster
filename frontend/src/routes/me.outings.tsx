import {createFileRoute, Link} from '@tanstack/react-router'
import {requireAuth, useMyOutings} from "@/queries.ts";
import {OutingCard} from "@/components/OutingCard.tsx";
import styles from "@/routes/me.outings.module.css"
import type {Outing} from "@/types.ts";
import {useState} from "react";



export const Route = createFileRoute('/me/outings')({
    beforeLoad: async ({context}) => requireAuth(context.queryClient),
  component: MeOutingsPage,
})

export function MeOutingsPage() {
    const {data, isPending, error} = useMyOutings()
    const [tab, setTab] = useState<Phase>("upcoming")

    if (isPending) return <div>Outings loading...</div>
    if (error) return <div>{error.message}</div>



    const all = [
        ...data.hosting.map(outing => ({outing, hosting: true})),
        ...data.joined.map(outing => ({outing, hosting: false})),
    ]

    const counts: Record<Phase, number> = {upcoming: 0, in_progress: 0, past: 0}
    const visible = all
        .filter(item => item.outing.phase === tab)
        .sort((a, b) => {
            const diff = new Date(a.outing.starts_at).getTime() - new Date(b.outing.starts_at).getTime()
            return tab === "past" ? -diff : diff
        })
    for (const item of all) counts[item.outing.phase]++
    return <div className={styles.page}>
        <h1 className={styles.heading}>My outings</h1>

        <div className={styles.tabs} role="tablist" aria-label="Outings by time">
            {TABS.map(t => (
                <button key={t.phase} type="button" role="tab" className={styles.tab}
                        aria-selected={tab === t.phase} onClick={() => setTab(t.phase)}>
                    {t.label}
                    <span className={`${styles.count} ${t.phase === "in_progress" && counts[t.phase] > 0 ? styles.countLive : ""}`}>
                {counts[t.phase]}
            </span>
                </button>
            ))}
        </div>

        {visible.length === 0
            ? <div className={styles.empty}>
                <p>{EMPTY[tab]}</p>
                {tab === "upcoming" && <Link className="btn-primary btn" to="/outings/new">Create outing</Link>}
            </div>
            : <div className={styles.list} role="tabpanel">
                {visible.map(({outing, hosting}) => (
                    <Link className={styles.cardLink} key={outing.id} to="/outings/$id" params={{id: outing.id}}>
                        <OutingCard outing={outing} label={roleLabel(hosting, outing.phase)}/>
                    </Link>
                ))}
                {tab === "in_progress" && <p className={styles.note}>
                    Once an outing starts, the group is locked: no new requests, changes or withdrawals. Comments and chat stay open.
                </p>}
            </div>
        }
    </div>

}

type Phase = Outing["phase"]

const TABS: {phase: Phase, label: string}[] = [
    {phase: "upcoming", label: "Upcoming"},
    {phase: "in_progress", label: "In progress"},
    {phase: "past", label: "Past"},
]

const EMPTY: Record<Phase, string> = {
    upcoming: "Nothing coming up. Host a hike or find one to join.",
    in_progress: "Nothing is happening right now. Outings show up here from their start time until they end.",
    past: "No past outings yet. The ones you host or join will be kept here.",
}

function roleLabel(hosting: boolean, phase: Phase): string {
    if (phase === "past") return hosting ? "You hosted" : "You went"
    return hosting ? "You're hosting" : "You're going"
}
