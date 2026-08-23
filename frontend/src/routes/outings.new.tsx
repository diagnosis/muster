import {createFileRoute, useNavigate} from '@tanstack/react-router'
import {requireAuth} from "@/queries.ts";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {CreateOutingInput, Outing} from "@/types.ts";

import {apiClient, ApiRequestError} from "@/lib/api.ts";
import {OutingForm} from "@/components/OutingForm.tsx";

export const Route = createFileRoute('/outings/new')({
    beforeLoad : async ({context}) => requireAuth(context.queryClient),
  component: CreateOutingPage,
})

export function CreateOutingPage() {

    const qc = useQueryClient()
    const navigate = useNavigate()


    const createMutation = useMutation({
        mutationFn: async (body:CreateOutingInput) => {
            const res = await apiClient.post<Outing>('/api/outings', body)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess:(data)=>{
            qc.invalidateQueries({queryKey:['outings']})
            navigate({to:'/outings/$id', params: {id: data.id}})
        }
    })



  return (
      <div>
        <OutingForm
            heading={"Create outing"}
            onSubmit={values => createMutation.mutate(values)}
            submitLabel={"Create outing"}
            error={createMutation.error instanceof ApiRequestError ? createMutation.error: undefined}
            pending={createMutation.isPending}
        />
      </div>
  )
}
