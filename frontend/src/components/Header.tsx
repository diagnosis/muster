
import {useMeQuery} from "../queries.ts";
import {Link, useNavigate} from "@tanstack/react-router";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import {apiClient} from "../lib/api.ts";


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
    if (isPending) return null
    if (data) return <div><span>{data.name}</span><span><button onClick={ () =>
       logout.mutate()
    }>Log out</button></span></div>
    return (
        <div>
            <Link to={'/login'}><button>Log in</button></Link>
            <Link to={'/signup'}><button>Sign up</button></Link>
        </div>
    )
}