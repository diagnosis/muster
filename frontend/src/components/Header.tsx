
import {useMeQuery} from "../queries.ts";
import {Link, useNavigate} from "@tanstack/react-router";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient} from "../lib/api.ts";
import styles from  "./Header.module.css"
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
            throw new Error(res.error.message)
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
        <div className={styles.nav}>
            <div className={styles.inner}>
                <Link className={styles.logo} to="/">Muster</Link>
            </div>
        </div>
    )
    return (
        <div className={styles.nav} ref={headerRef}>
            <div className={styles.inner}>
            <Link className={styles.logo} to={'/'} onClick={()=>setOpen(false)}>Muster</Link>
                <button className={`${styles.toggle} ${styles.hamburgerBtn}`} aria-expanded={open} onClick={()=> setOpen(o => !o)}>☰</button>
            <div className={`${styles.panel} ${open ? styles.panelOpen : ""}`}>
                {data ? (
                        <div className={styles.userOutings}>
                            <Link to="/me/outings" onClick={()=> setOpen(false)}>my outings</Link>
                            <Link to="/outings/new" onClick={() => setOpen(false)}>Create new</Link>
                            <div className={styles.loginSignup}>
                                <Link onClick={()=>setOpen(false)} to={"/me/profile"}>{data.name}</Link>
                                <button onClick={ () =>
                                    logout.mutate()
                                }>Log out</button>
                            </div>
                        </div>

                    ) :
                    (<div className={styles.loginSignup}>
                        <Link to={'/login'} onClick={()=>setOpen(false)}><button>Log in</button></Link>
                        <Link to={'/signup'} onClick={() => setOpen(false)}><button>Sign up</button></Link>
                    </div>)
                }
            </div>
            </div>
        </div>
    )
}