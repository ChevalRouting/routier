import { ON_COLOR, PRIMARY, PRIMARY_FOREGROUND, TOPO } from '@/lib/palette'
import { BaseEdge, getBezierPath, Handle, Position, useReactFlow, type EdgeProps, type NodeProps } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useEffect } from 'react'

type DShape = { label: React.ReactNode; style?: string; selected?: boolean }

type FitViewOnChangeShape = { version: number }

type ParallelEdgeShape = { offset?: number }

export const S: Record<string, React.CSSProperties> = {
  router:    { background: PRIMARY, color: PRIMARY_FOREGROUND, border: 'none', borderRadius: 8, padding: '10px 18px', fontWeight: 700, fontSize: 13 },
  bgp:       { background: TOPO.bgp, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  static:    { background: TOPO.static, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  ospf:      { background: TOPO.ospf, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  ospf6:     { background: TOPO.ospf6, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  anycast:   { background: TOPO.anycast, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  tunnel:    { background: TOPO.tunnel, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  wireguard: { background: TOPO.wireguard, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  vrrp:      { background: TOPO.vrrp, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
  vrf:       { background: TOPO.vrf, color: ON_COLOR, border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content' },
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
  const d = data as DShape
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

export function ParallelEdge({ sourceX, sourceY, targetX, targetY, sourcePosition, targetPosition, data, style, markerEnd }: EdgeProps) {
  const offset = (data as ParallelEdgeShape | undefined)?.offset ?? 0
  const horizontal = sourcePosition === Position.Left || sourcePosition === Position.Right
  const dx = horizontal ? 0 : offset
  const dy = horizontal ? offset : 0
  const [path] = getBezierPath({ sourceX: sourceX + dx, sourceY: sourceY + dy, sourcePosition, targetX: targetX + dx, targetY: targetY + dy, targetPosition })
  return <BaseEdge path={path} style={style} markerEnd={markerEnd} />
}

export const edgeTypes = { parallel: ParallelEdge }

export function FitViewOnChange({ version }: FitViewOnChangeShape) {
  const { fitView } = useReactFlow()
  useEffect(() => {
    fitView({ padding: 0.25, duration: 250 })
  }, [version, fitView])
  return null
}
