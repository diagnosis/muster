// src/routes/__root.tsx

import {Outlet, createRootRouteWithContext} from '@tanstack/react-router'
import type {QueryClient} from "@tanstack/react-query";

export const Route =
    createRootRouteWithContext<{queryClient:QueryClient}>()({
        component: () => <Outlet/>,
    })


