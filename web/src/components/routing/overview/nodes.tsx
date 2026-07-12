import { useEffect } from 'react'
import { TOPO } from '@/lib/palette'
import { Handle, Position, useReactFlow, type NodeProps } from '@xyflow/react'
import '@xyflow/react/dist/style.css'

export const S: Record<string, React.CSSProperties> = {
  router:    { background: 'hsl(var(--primary))', color: 'hsl(var(--primary-foreground))', border: 'none', borderRadius: 8, padding: '10px 18px', fontWeight: 700, fontSize: 13 },
  bgp:       { background: TOPO.bgp, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  static:    { background: TOPO.static, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  ospf:      { background: TOPO.ospf, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  ospf6:     { background: TOPO.ospf6, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  anycast:   { background: TOPO.anycast, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  tunnel:    { background: TOPO.tunnel, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  wireguard: { background: TOPO.wireguard, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  vrrp:      { background: TOPO.vrrp, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  vrf:       { background: TOPO.vrf, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
}

export const HH: React.CSSProperties = { opacity: 0, width: 8, height: 8, pointerEvents: 'none' }

export function RouterNode({ data }: NodeProps) {
  return (
    <div style={S.router}>
      {(['Left','Right','Top','Bottom'] as const).map((p) => (
        <>
          <Handle key={`s-${p}`} type="source" position={Position[p]} id={p.toLowerCase()} style={HH} />
          <Handle key={`t-${p}`} type="target" position={Position[p]} id={`${p.toLowerCase()}-t`} style={HH} />
        </>
      ))}
      {String(data.label)}
    </div>
  )
}

export function GroupNode({ data }: NodeProps) {
  const d = data as { label: React.ReactNode; style?: string; selected?: boolean }
  const base = S[d.style ?? ''] ?? S.bgp
  return (
    <div style={{ ...base, outline: d.selected ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' }}>
      {(['Left','Right','Top','Bottom'] as const).map((p) => (
        <>
          <Handle key={`s-${p}`} type="source" position={Position[p]} id={p.toLowerCase()} style={HH} />
          <Handle key={`t-${p}`} type="target" position={Position[p]} id={`${p.toLowerCase()}-t`} style={HH} />
        </>
      ))}
      {d.label}
    </div>
  )
}

export const nodeTypes = { router: RouterNode, group: GroupNode }
export function FitViewOnChange({ version }: { version: number }) {
  const { fitView } = useReactFlow()
  useEffect(() => {
    fitView({ padding: 0.25, duration: 250 })
  }, [version, fitView])
  return null
}
