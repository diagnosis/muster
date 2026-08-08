// src/routes/__root.tsx

import {Outlet, createRootRouteWithContext} from '@tanstack/react-router'
import type {QueryClient} from "@tanstack/react-query";
import {Header} from "../components/Header.tsx";
import styles from "./Layout.module.css"

export const Route =
    createRootRouteWithContext<{queryClient:QueryClient}>()({
        component: RouteComponent,
    })

function RouteComponent(){
    return <>
        <Header/>
        <div className={styles.shell}>
            <Outlet/>
        </div>
    </>
}

