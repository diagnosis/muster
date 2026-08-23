import {createFileRoute, redirect, useNavigate} from '@tanstack/react-router'
import {meQueryOptions, outingQueryOptions, useOuting} from "@/queries.ts";
import {OutingForm} from "@/components/OutingForm.tsx";
import {useMutation, useQueryClient} from "@tanstack/react-query";
import type {CreateOutingInput, Outing} from "@/types.ts";
import {apiClient, ApiRequestError} from "@/lib/api.ts";

export const Route = createFileRoute('/outings_/$id/edit')({
    beforeLoad: async ({context, params}) => {
        const me = await context.queryClient.ensureQueryData(meQueryOptions())
        if (!me) throw redirect({ to: '/login' })

        const detail = await context.queryClient.ensureQueryData(outingQueryOptions(params.id))
        if (detail.outing.host_id !== me.id){
            throw redirect({to:'/outings/$id', params:{id:params.id}})
        }
    },
    component: EditOutingPage,
})

export function EditOutingPage() {
    const {id} = Route.useParams()
    const qc = useQueryClient()
    const navigate = useNavigate()
    const {data: detail} = useOuting(id)


    const editMutation = useMutation({
        mutationFn: async (body:CreateOutingInput) => {
            const res = await apiClient.patch<Outing>(`/api/outings/${id}`, body)
            if (res.ok){
                return res.data
            }
            throw new ApiRequestError(res.error, res.httpStatus)
        },
        onSuccess : () => {
            qc.invalidateQueries({queryKey:['outing', id]})
            qc.invalidateQueries({queryKey:['outings']})
            qc.invalidateQueries({queryKey:['my-outings']})
            navigate({to:'/outings/$id', params: {id: id}})
        }
    })

    if (!detail){
        return null
    }

  return <div>
      <OutingForm heading={"Edit outing"}
                  onSubmit={values => editMutation.mutate(values)}
                  submitLabel={"Save changes"}
                  error={editMutation.error instanceof ApiRequestError ? editMutation.error : undefined}
                  pending={editMutation.isPending}
                  initial={detail.outing}
      />

  </div>
}
