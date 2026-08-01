import {useEffect, useState} from 'react'
import type {Outing} from './types'
import { apiClient } from './lib/api'
import './App.css'

function App() {
  const [outings, setOutings] = useState<Outing[]>([])
  const [error, setError] =useState<string|null>(null)

  useEffect( () => {
    const load = async ()  => {
      const res = await apiClient.get<Outing[]>('/api/outings')
        if (res.ok){
            setOutings(res.data)
        }else{
            setError(res.error.message)
        }
    }
   load()

  }, []);

  if (error){
    return <div>
      error loading outings: {error}
    </div>
  }else{
    return <>
      <ul>
        <h2>Outings</h2>
        {outings.map(outing => <li key={outing.id}>{outing.title}</li>)}
      </ul>

    </>
  }
}
export default App
