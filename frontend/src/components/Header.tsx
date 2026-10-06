
import {invalidateFresh, useMeQuery} from "@/queries.ts";
import {Link, useNavigate} from "@tanstack/react-router";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import styles from  "@/components/Header.module.css"
import {useEffect, useRef, useState} from "react";
import {NotificationBell} from "@/components/NotificationBell.tsx";
import {MenuIcon, MessageIcon} from "@/components/Icons.tsx";
import {useConversations} from "@/queries/conversation.ts";
import {EVENT_TYPES, useEvents} from "@/events/EventProvider.tsx";



export function Header(){
    const [open, setOpen] = useState(false)
    const {data, isPending} = useMeQuery()
    const queryClient = useQueryClient()
    const navigate = useNavigate()
    const headerRef = useRef<HTMLDivElement>(null)

    const logout = useMutation({
        mutationFn: async () => {
            const res = await apiClient.post("/api/auth/logout")
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSettled : () => {
            queryClient.invalidateQueries({queryKey:['me']})
            navigate({to:'/'})
            setOpen(false)
        }
    })
    useEffect(()=>{
        const handleOutsideClick = (event: MouseEvent) =>{
            if (headerRef.current && event.target instanceof Node && !headerRef.current.contains(event.target)){
                setOpen(false)
            }
        };
        if(open){
            document.addEventListener("mousedown", handleOutsideClick)
        }

        //clean up
        return () => {
            document.removeEventListener("mousedown", handleOutsideClick)
        }
    },[open])

    if (isPending) return (
        <header className={styles.nav}>
            <div className={styles.inner}>
                <Link className={styles.logo} to="/">Muster</Link>
            </div>
        </header>
    )

    return (
        <header className={styles.nav} ref={headerRef}>
            <div className={styles.inner}>
            <Link className={styles.logo} to={'/'} onClick={()=>setOpen(false)}>Muster</Link>
                {data ? (
                    <div className={styles.headerActions}>
                        <InboxEvents/>
                        <InboxIconLink/>
                        <span className={styles.mobileBell}><NotificationBell/></span>
                        <button
                            aria-label={'Menu'}
                            className={`${styles.toggle} ${styles.hamburgerBtn}`}
                            aria-expanded={open} onClick={()=> setOpen(o => !o)}><MenuIcon/></button>
                    </div>

                ): <button
                    aria-label={'Menu'}
                    className={`${styles.toggle} ${styles.hamburgerBtn}`}
                    aria-expanded={open} onClick={()=> setOpen(o => !o)}><MenuIcon/></button>}

            <div className={`${styles.panel} ${open ? styles.panelOpen : ""}`}>
                {data ? (
                        <div className={styles.userOutings}>
                            <Link className={styles.navLink} to="/me/outings" onClick={()=> setOpen(false)}>My outings</Link>
                            <InboxLink onNavigate={()=>setOpen(false)}/>
                            <Link className={`btn btn-primary ${styles.navCta}`} to="/outings/new" onClick={() => setOpen(false)}>Create outing</Link>
                            <span className={styles.desktopBell}><NotificationBell/></span>
                            <Link className={`${styles.navLink} ${styles.profileLink}`} onClick={() => setOpen(false)} to={"/me/profile"}>
                                <span className={styles.avatar} aria-hidden="true">{data.name.charAt(0).toUpperCase()}</span>
                                <span className={styles.profileName}>{data.name}</span>
                            </Link>
                            <div className={styles.loginSignup}>
                                <button className={styles.logoutBtn} onClick={ () =>
                                    logout.mutate()
                                }>Log out</button>
                            </div>
                        </div>

                    ) :
                    (<div className={styles.loginSignup}>
                        <Link className={'btn'} to={'/login'} onClick={()=>setOpen(false)}>Log in</Link>
                        <Link className={'btn-primary btn'} to={'/signup'} onClick={() => setOpen(false)}>Sign up</Link>
                    </div>)
                }
            </div>
            </div>
        </header>
    )
}

// Renders nothing. Keeps the inbox data fresh on every page while logged in.
function InboxEvents() {
    const qc = useQueryClient()
    const {subscribe} = useEvents()
    useEffect(() => {
        const onPoke = () => { invalidateFresh(qc, {queryKey: ['conversations']}) }
        const offs = EVENT_TYPES.map(type => subscribe(type, onPoke))
        return () => offs.forEach(off => off())
    }, [subscribe, qc])
    return null
}

function useUnreadTotal() {
    const {data} = useConversations()
    return data?.conversations.reduce((sum, c) => sum + c.unread_count, 0) ?? 0
}

// the text link, in the desktop bar and inside the phone menu
function InboxLink({onNavigate}: {onNavigate: () => void}) {
    const unread = useUnreadTotal()
    return (
        <Link className={`${styles.navLink} ${styles.inboxText}`} to="/inbox" onClick={onNavigate}>
            Inbox
            {unread > 0 && <span className={styles.unread}>{unread}</span>}
        </Link>
    )
}

// the icon, in the phone's top bar next to the bell
function InboxIconLink() {
    const unread = useUnreadTotal()
    return (
        <Link className={styles.inboxIcon} to="/inbox">
            <MessageIcon size={22}/>
            <span className={styles.srOnly}>Inbox</span>
            {unread > 0 && <span className={styles.iconBadge}>{unread}</span>}
        </Link>
    )
}