import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'

export function ReloadButton({ onClick }: { onClick: () => void }) {
  return (
    <Button variant="outline" size="sm" onClick={onClick} className="gap-2">
      <RefreshCw className="h-4 w-4" />Reload
    </Button>
  )
}
