
import {useMeQuery} from "../queries.ts";
import {Link, useNavigate} from "@tanstack/react-router";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient} from "../lib/api.ts";
import styles from  "./Header.module.css"


export function Header(){
    const {data, isPending} = useMeQuery()
    const queryClient = useQueryClient()
    const navigate = useNavigate()
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
        }
    })
    if (isPending) return (
        <div className={styles.nav}>
            <Link className={styles.logo} to="/">Muster</Link>
        </div>
    )
    return (
        <div className={styles.nav}>
            <Link className={styles.logo} to={'/'}>Muster</Link>
            {data ? (
                <div className={styles.userOutings}>
                    <Link to="/me/outings">my outings</Link>
                    <div className={styles.loginSignup}>
                        <span>{data.name}</span>
                        <button onClick={ () =>
                            logout.mutate()
                        }>Log out</button>
                    </div>
                </div>

            ) :
                (<div className={styles.loginSignup}>
                <Link to={'/login'}><button>Log in</button></Link>
                <Link to={'/signup'}><button>Sign up</button></Link>
            </div>)
            }
        </div>
    )
}