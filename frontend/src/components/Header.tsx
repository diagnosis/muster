
import {useMeQuery} from "@/queries.ts";
import {Link, useNavigate} from "@tanstack/react-router";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient, ApiRequestError} from "@/lib/api.ts";
import styles from  "@/components/Header.module.css"
import {useEffect, useRef, useState} from "react";


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
                <button
                    aria-label={'Menu'}
                    className={`${styles.toggle} ${styles.hamburgerBtn}`}
                    aria-expanded={open} onClick={()=> setOpen(o => !o)}>☰</button>
            <div className={`${styles.panel} ${open ? styles.panelOpen : ""}`}>
                {data ? (
                        <div className={styles.userOutings}>
                            <Link className={styles.navLink} to="/me/outings" onClick={()=> setOpen(false)}>My outings</Link>
                            <Link className={styles.navLink} to="/outings/new" onClick={() => setOpen(false)}>Create outing</Link>
                            <Link className={styles.navLink} onClick={()=>setOpen(false)} to={"/me/profile"}>{data.name}</Link>
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