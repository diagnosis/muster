// src/routes/__root.tsx

import {Outlet, createRootRouteWithContext} from '@tanstack/react-router'
import type {QueryClient} from "@tanstack/react-query";
import {Header} from "../components/Header.tsx";

export const Route =
    createRootRouteWithContext<{queryClient:QueryClient}>()({
        component: RouteComponent,
    })

function RouteComponent(){
    return <>
        <Header/>
        <Outlet/>
    </>
}

