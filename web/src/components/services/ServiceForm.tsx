import { ConfigFileRow, ServiceConfig, ServiceFormProps, newConfigFileId } from '@/components/services/shared'
import { Input, Button, EntryRow, PreferencesGroup, SwitchRow } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'

export function ServiceForm({ service, rows, onServiceChange, onRowsChange, nameInput, onNameChange, onAdd, onDone }: ServiceFormProps) {
  const set = <K extends keyof ServiceConfig>(key: K, val: ServiceConfig[K]) =>
    onServiceChange({ ...service, [key]: val })

  const addRow = () => onRowsChange([...rows, { id: newConfigFileId(), src: '', dest: '', mode: 644 }])
  const updRow = (id: number, patch: Partial<ConfigFileRow>) =>
    onRowsChange(rows.map((r) => (r.id === id ? { ...r, ...patch } : r)))
  const delRow = (id: number) => onRowsChange(rows.filter((r) => r.id !== id))

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        {onNameChange !== undefined && (
          <EntryRow
            title="Service name"
            autoFocus
            value={nameInput ?? ''}
            onChange={(e) => onNameChange(e.target.value)}
            placeholder="nginx"
            className="font-mono"
          />
        )}
        <SwitchRow
          title="Enabled"
          subtitle="Start on boot and keep running"
          checked={service.enable}
          onCheckedChange={(v) => set('enable', v)}
        />
        <EntryRow
          title="After"
          value={service.after}
          onChange={(e) => set('after', e.target.value)}
          placeholder="network"
          className="font-mono"
        />
      </PreferencesGroup>

      <PreferencesGroup
        title="Config files"
        description="Files deployed to the target when this service is applied."
        header={
          <Button variant="outline" size="sm" onClick={addRow} className="gap-1.5">
            <Plus className="h-4 w-4" />Add
          </Button>
        }
      >
        {rows.length === 0 ? (
          <p className="px-4 py-3 text-sm text-muted-foreground">No config files, this service runs without any file deployments.</p>
        ) : (
          rows.map((row) => (
            <div key={row.id} className="flex items-center gap-2 px-4 py-2">
              <Input
                value={row.src}
                onChange={(e) => updRow(row.id, { src: e.target.value })}
                placeholder="/templates/nginx.conf"
                className="min-w-0 flex-1 bg-transparent font-mono text-xs text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <span className="shrink-0 text-muted-foreground">→</span>
              <Input
                value={row.dest}
                onChange={(e) => updRow(row.id, { dest: e.target.value })}
                placeholder="/etc/nginx/nginx.conf"
                className="min-w-0 flex-1 bg-transparent font-mono text-xs text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <Input
                value={row.mode}
                onChange={(e) => updRow(row.id, { mode: Number(e.target.value) })}
                placeholder="644"
                className="w-12 shrink-0 bg-transparent font-mono text-xs text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
                onClick={() => delRow(row.id)}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          ))
        )}
      </PreferencesGroup>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add service</Button>}
      </div>
    </div>
  )
}
