import { useParams } from 'react-router-dom'
import { SwitchDetailView } from '@/components/switches/SwitchDetailView'

export default function SwitchDetail() {
  const { name = '' } = useParams()
  return <SwitchDetailView key={name} name={name} />
}
