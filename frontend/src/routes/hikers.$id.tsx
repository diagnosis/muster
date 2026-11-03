import {createFileRoute, Link, useNavigate} from '@tanstack/react-router'
import {useHiker} from "@/queries/profile.ts";
import {requireAuth, useMeQuery} from "@/queries.ts";
import type {PublicProfile} from "@/types.ts";
import {useStartDM} from "@/queries/conversation.ts";
import styles from "@/routes/hikers.$id.module.css"
import {initial} from "@/lib/format.ts";
import {MessageIcon} from "@/components/Icons.tsx";
import {memberSince} from "@/utils/date.ts";

export const Route = createFileRoute('/hikers/$id')({
  beforeLoad : async ({context}) => requireAuth(context.queryClient),
  component: PublicProfilePage,
})

function PublicProfilePage() {
  const {id} = Route.useParams()
  const {data: hiker, isPending, error} = useHiker(id)
  const {data:me} = useMeQuery()
  if (isPending) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>
  if (hiker === null) return <div>No such hiker.</div>
  return <div><ProfileCard hiker={hiker} isMe={me?.id === hiker.id}/></div>
}
interface ProfileCardProps {
  hiker: PublicProfile
  isMe: boolean
}

function ProfileCard({hiker, isMe}: ProfileCardProps) {
  const navigate = useNavigate()
  const startDM = useStartDM()
  const subtitle = [hiker.experience, hiker.home_area].filter(Boolean).join(' · ')

  return (
      <article className={styles.card}>
        <header className={styles.header}>
          <span className={styles.avatar} aria-hidden="true">{initial(hiker.name)}</span>
          <div className={styles.identity}>
            <h1 className={styles.name}>{hiker.name}</h1>
            <p className={styles.subtitle}>{subtitle}</p>
          </div>
          {isMe
              ? <Link className="btn btn-outline" to="/me/profile">Edit profile</Link>
              : <button type="button" className={`btn btn-outline ${styles.message}`}
                        disabled={startDM.isPending}
                        onClick={() => startDM.mutate(hiker.id, {
                          onSuccess: (conv) => navigate({to: '/conversations/$id', params: {id: conv.id}})
                        })}>
                <MessageIcon/> Message
              </button>}
        </header>

        {hiker.bio && <p className={styles.bio}>{hiker.bio}</p>}

        <dl className={styles.stats}>
          <div className={styles.stat}>
            <dt>Hosted</dt>
            <dd>{hiker.hosted_count}</dd>
          </div>
          <div className={styles.stat}>
            <dt>Went on</dt>
            <dd>{hiker.went_count}</dd>
          </div>
          <div className={styles.stat}>
            <dt>Hiking with Muster since</dt>
            <dd>{memberSince(hiker.member_since)}</dd>
          </div>
        </dl>
        {startDM.isError && <p className={styles.error}>{startDM.error.message}</p>}
      </article>
  )
}