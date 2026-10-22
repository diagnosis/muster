// src/routes/__root.tsx

import {Outlet, createRootRouteWithContext, useNavigate} from '@tanstack/react-router'
import type {QueryClient} from "@tanstack/react-query";
import {Header} from "@/components/Header.tsx";
import styles from "@/routes/Layout.module.css"
import {NotFound} from "@/components/NotFound.tsx";
import {EventsProvider} from "@/events/EventProvider.tsx";
import {forgetUserData, useMeQuery} from "@/queries.ts";
import {useEffect} from "react";
import {setOnSignedOut} from "@/lib/api.ts";

export const Route =
    createRootRouteWithContext<{queryClient:QueryClient}>()({

        notFoundComponent:NotFound,
        component: RouteComponent,
    })

export function RouteComponent(){
    const { data: me } = useMeQuery()
    const { queryClient } = Route.useRouteContext()
    const navigate = useNavigate()
    useEffect(() => {
        setOnSignedOut(() => {
            queryClient.setQueryData(['me'], null)
            forgetUserData(queryClient)
            navigate({to: '/login'})
        })
    }, [queryClient, navigate])
    return <>
        <EventsProvider enabled={!!me}>
            <Header/>
            <div className={styles.shell}>
                <Outlet/>
            </div>
        </EventsProvider>
    </>
}

