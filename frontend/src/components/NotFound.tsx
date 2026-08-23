import styles from "@/components/NotFound.module.css"
import {Link} from "@tanstack/react-router";

export function NotFound(){
    return <div className={styles.container}>
        <div className={styles.content}>
            <h1 className={styles.title}>Dead end</h1>
            <p className={styles.description}>This trail doesn't go anywhere. The outing may have been cancelled, or the link you followed is wrong.</p>
        </div>

       <Link to={"/"} className={"btn btn-primary"}>Back to outings</Link>
    </div>
}