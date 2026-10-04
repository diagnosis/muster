
import {useMeQuery} from "@/queries.ts";
import {Link, useNavigate} from "@tanstack/react-router";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import styles from  "@/components/Header.module.css"
import {useEffect, useRef, useState} from "react";
import {NotificationBell} from "@/components/NotificationBell.tsx";
import {MenuIcon} from "@/components/Icons.tsx";


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
                            <Link className={styles.navLink} to="/inbox" onClick={() => setOpen(false)}>Inbox</Link>
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