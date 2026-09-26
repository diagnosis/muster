// src/routes/__root.tsx

import {Outlet, createRootRouteWithContext} from '@tanstack/react-router'
import type {QueryClient} from "@tanstack/react-query";
import {Header} from "@/components/Header.tsx";
import styles from "@/routes/Layout.module.css"
import {NotFound} from "@/components/NotFound.tsx";
import {EventsProvider} from "@/events/EventProvider.tsx";
import {useMeQuery} from "@/queries.ts";

export const Route =
    createRootRouteWithContext<{queryClient:QueryClient}>()({
        notFoundComponent:NotFound,
        component: RouteComponent,
    })

export function RouteComponent(){
    const { data: me } = useMeQuery()
    return <>
        <EventsProvider enabled={!!me}>
            <Header/>
            <div className={styles.shell}>
                <Outlet/>
            </div>
        </EventsProvider>
    </>
}

