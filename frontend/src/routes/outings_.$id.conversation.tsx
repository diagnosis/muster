import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/outings_/$id/conversation')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/outings_$id/conversation"!</div>
}
