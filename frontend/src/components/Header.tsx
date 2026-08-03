
import {useMeQuery} from "../queries.ts";


export function Header(){
    const {data, isPending} = useMeQuery()
    if (isPending) return null
    if (data) return <div><span>{data.name}</span></div>
    return (
        <div>
            <button>Log in</button>
            <button>Sign up</button>
        </div>
    )
}