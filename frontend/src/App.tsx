import type {Outing} from './types'
import { apiClient } from './lib/api'
import './App.css'
import {OutingCard} from "./components/OutingCard.tsx";
import {useQuery} from "@tanstack/react-query";

function App() {


  async function getOutings(){
    const res = await apiClient.get<Outing[]>('/api/outings')
    if (res.ok){
      return res.data
    }
    throw new Error(res.error.message)
  }
  //using tanstack query
  const {data: outings, isPending, error} = useQuery(
      {
        queryKey:['outings'],
        queryFn: getOutings
      }
  )
  if (isPending){
    return <div>Loading outings...</div>
  }

  if (error){
    return <div>
      error loading outings: {error.message}
    </div>
  }
  return <>
    <div className='wrapper'>
      <h2>Outings</h2>
      {outings.map(outing=><OutingCard key={outing.id} outing={outing}/>)}
    </div>
    </>

}
export default App
