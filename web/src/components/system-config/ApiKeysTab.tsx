import type { TypesAPIKey } from '@/api'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { AlertDialog, Button, CopyButton, Dialog, EmptyState, Input, SectionLabel, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
import { KeyRound, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

type ApiKeysTabShape = { name: string; token: string }

export function ApiKeysTab() {
  const { data, isLoading, reload } = useFetch(() => api.apiAuthApiKeysGet())
  const [name, setName] = useState('')
  const [creating, setCreating] = useState(false)
  const [created, setCreated] = useState<ApiKeysTabShape | null>(null)
  const [revoke, setRevoke] = useState<TypesAPIKey | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  const keys = (data ?? []).filter((k) => !k.revoked_at)

  const create = () => {
    const trimmed = name.trim()
    if (!trimmed) return
    setCreating(true)
    api.apiAuthApiKeysPost({ TypesCreateAPIKeyRequest: { name: trimmed } })
      .then((res) => {
        const token = res.token ?? ''
        setCreated({ name: trimmed, token })
        setName('')
        reload()
      })
      .catch((e) => toast.error(`Create failed: ${e.message}`))
      .finally(() => setCreating(false))
  }

  const doRevoke = () => {
    const k = revoke
    if (!k) return
    setRevoke(null)
    setBusy(k.id)
    api.apiAuthApiKeysIdDelete({ id: k.id })
      .then(() => { toast.success('API key revoked'); reload() })
      .catch((e) => toast.error(`Revoke failed: ${e.message}`))
      .finally(() => setBusy(null))
  }

  return (
    <div className="space-y-4">
      <SectionLabel>API Keys</SectionLabel>
      <p className="text-sm text-muted-foreground">
        API keys authenticate programmatic clients such as <code className="text-xs bg-muted px-1 py-0.5 rounded">routier-mcp</code> using a
        <code className="text-xs bg-muted px-1 py-0.5 rounded">Bearer</code> token. They cannot manage authentication credentials. The full token is
        shown only once at creation.
      </p>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') create() }}
          placeholder="Key name (e.g. edge-a-mcp)"
          className="max-w-xs"
          disabled={creating}
        />
        <Button onClick={create} disabled={creating || !name.trim()} className="gap-2">
          {creating ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}Generate key
        </Button>
      </div>

      {isLoading ? <Spinner /> : keys.length === 0 ? (
        <EmptyState
          icon={<KeyRound />}
          title="No API keys"
          message="Generate a key to authenticate programmatic clients."
        />
      ) : (
        <Table>
          <TableHeader>
            <TableRow><TableHead>Name</TableHead><TableHead>Prefix</TableHead><TableHead>Created</TableHead><TableHead>Created by</TableHead><TableHead></TableHead></TableRow>
          </TableHeader>
          <TableBody>
            {keys.map((k) => (
              <TableRow key={k.id}>
                <TableCell>{k.name}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{k.prefix}…</TableCell>
                <TableCell className="font-mono text-xs">{new Date(k.created_at * 1000).toLocaleString()}</TableCell>
                <TableCell className="text-muted-foreground">{k.created_by}</TableCell>
                <TableCell className="text-right">
                  <Button variant="outline" size="sm" className="gap-1.5" disabled={busy === k.id} onClick={() => setRevoke(k)}>
                    <Trash2 className="h-3.5 w-3.5" />Revoke
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Dialog open={!!created} onClose={() => setCreated(null)} title="API key created">
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Copy the token for <span className="font-medium text-foreground">{created?.name}</span> now. It will not be shown again.
          </p>
          <div className="flex items-center gap-2 rounded-md bg-[#09090b] p-3">
            <code className="flex-1 overflow-auto whitespace-nowrap font-mono text-xs text-zinc-200">{created?.token}</code>
            <CopyButton text={created?.token ?? ''} />
          </div>
        </div>
      </Dialog>

      <AlertDialog
        open={!!revoke}
        onCancel={() => setRevoke(null)}
        onConfirm={doRevoke}
        title="Revoke this API key?"
        description={revoke ? `Clients using "${revoke.name}" will immediately lose access. This cannot be undone.` : undefined}
        confirmLabel="Revoke"
        cancelLabel="Cancel"
        destructive
      />
    </div>
  )
}
