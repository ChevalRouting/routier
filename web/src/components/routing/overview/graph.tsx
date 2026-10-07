import type { TypesKernelRoute as KernelRoute, TypesOSPFNeighborSummary as OSPFNeighbor, TypesVRRPInstanceStatus as VRRPInstanceStatus } from '@/api'
import { protoColor } from '@/lib/palette'
import { type Edge, type Node } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { S } from './nodes'
import { OSPFArea, RoutingConfig, Tunnel, WgIface } from './types'

type FromShape = { x: number; y: number }

type ToShape = { x: number; y: number }

type VrfDeclarationsShape = { table: number }

type BuildGraphShape = { nodes: Node[]; edges: Edge[] }

type SrcShape = { x: number; y: number }

type DstShape = { x: number; y: number }

export function edgeHandles(from: FromShape, to: ToShape) {
  const dx = to.x - from.x
  const dy = to.y - from.y
  if (Math.abs(dx) >= Math.abs(dy)) {
    return dx > 0
      ? { sourceHandle: 'right', targetHandle: 'left-t' }
      : { sourceHandle: 'left', targetHandle: 'right-t' }
  }
  return dy > 0
    ? { sourceHandle: 'bottom', targetHandle: 'top-t' }
    : { sourceHandle: 'top', targetHandle: 'bottom-t' }
}

export function normalizeAreaId(id: string): string {
  if (id.includes('.')) return id
  const n = parseInt(id, 10)
  if (isNaN(n)) return id
  return `${(n >>> 24) & 0xff}.${(n >>> 16) & 0xff}.${(n >>> 8) & 0xff}.${n & 0xff}`
}

export const RX = 650
export const RY = 320

export const VRRP_STATE_COLOR: Record<string, string> = {
  MASTER:  '#16a34a',
  BACKUP:  '#dc2626',
  FAULT:   '#dc2626',
  UNKNOWN: '#6b7280',
}
export const OSPF_STATE_COLOR: Record<string, string> = {
  'Full/DR':    '#16a34a',
  'Full/BDR':   '#16a34a',
  'Full/Other': '#16a34a',
  '2-Way/Other':'#d97706',
}

export interface GraphInput {
  routing: RoutingConfig
  tunnels: Record<string, Tunnel>
  wireguard: Record<string, WgIface>
  hostname: string
  ospfNeighbors: OSPFNeighbor[]
  vrrpInstances: VRRPInstanceStatus[]
  kernelRoutes: KernelRoute[]
  vrfDeclarations: Record<string, VrfDeclarationsShape>
  selectedProto: string | null
}

export function buildGraph({ routing, tunnels, wireguard, hostname, ospfNeighbors, vrrpInstances, kernelRoutes, vrfDeclarations, selectedProto }: GraphInput): BuildGraphShape {
  const nodes: Node[] = []
  const edges: Edge[] = []

  const push = (n: Node) => nodes.push(n)
  const sel  = (id: string) => id === selectedProto

  function connect(id: string, src: SrcShape, dst: DstShape, extra: Partial<Edge> = {}) {
    const sep = id.indexOf(':')
    const srcId = id.slice(0, sep)
    const dstId = id.slice(sep + 1)
    const h = edgeHandles(src, dst)
    edges.push({ id, source: srcId, target: dstId, ...h, ...extra })
  }

  const routerPos = { x: RX, y: RY }
  push({ id: 'router', type: 'router', data: { label: hostname || 'router' }, position: routerPos })

  const DEFAULT_DSTS = new Set(['default', '0.0.0.0/0', '::/0'])
  interface RouteItem { id: string; dst: string; via?: string; dev?: string; proto?: string; isKernel: boolean }
  const routeItems: RouteItem[] = []

  for (const [i, r] of (routing.static ?? []).slice(0, 10).entries()) {
    routeItems.push({ id: `route-static-${i}`, dst: r.destination, via: r.via, dev: r.dev, isKernel: false })
  }

  const normDst = (d: string) => d === 'default' ? '0.0.0.0/0' : d
  const configDsts = new Set((routing.static ?? []).map((r) => normDst(r.destination)))
  for (const r of kernelRoutes) {
    const dst = normDst(r.dst)
    if (!DEFAULT_DSTS.has(dst)) continue
    if (configDsts.has(dst)) continue
    const rid = `route-kernel-${dst.replace(/[:/]/g, '_')}-${r.protocol}`
    if (routeItems.some((x) => x.id === rid)) continue
    routeItems.push({ id: rid, dst, via: r.gateway, dev: r.dev, proto: r.protocol, isKernel: true })
  }

  if (routeItems.length > 0) {
    const routesPos = { x: RX - 40, y: RY - 200 }
    push({
      id: 'static',
      type: 'group',
      data: { label: `Routes (${routeItems.length})`, style: 'static', selected: sel('static'), nodeTitle: 'Routes' },
      position: routesPos,
      style: { ...S.static, outline: sel('static') ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' },
    })
    connect('router:static', routerPos, routesPos, { style: { stroke: '#16a34a' } })

    const cols = Math.min(routeItems.length, 5)
    const spacing = 200
    const startX = RX - ((cols - 1) * spacing) / 2
    routeItems.forEach((r, i) => {
      const childPos = { x: startX + (i % cols) * spacing, y: RY - 390 - Math.floor(i / cols) * 90 }
      const routeProtoColor = r.proto ? protoColor(r.proto) : '#16a34a'
      push({
        id: r.id,
        type: 'group',
        data: {
          label: (
            <div className="text-center leading-tight">
              <div className="font-bold font-mono text-[11px]">{r.dst}</div>
              {r.via && <div className="opacity-80 text-[10px]">via {r.via}</div>}
              {r.dev && !r.via && <div className="opacity-80 text-[10px]">dev {r.dev}</div>}
              {r.proto && (
                <div className="mt-0.5">
                  <span className="inline-block px-1 py-0.5 rounded text-white text-[9px]" style={{ background: routeProtoColor }}>{r.proto}</span>
                </div>
              )}
            </div>
          ),
          style: 'static',
          selected: sel(r.id),
          nodeTitle: r.dst,
        },
        position: childPos,
        style: { ...S.static, outline: sel(r.id) ? '2px solid white' : 'none', outlineOffset: 2 },
      })
      connect(`static:${r.id}`, routesPos, childPos, { style: { stroke: '#16a34a', opacity: 0.6 } })
    })
  }

  const services = routing.anycast?.services ?? []
  const leftSlots: Array<'bgp' | 'ospf' | 'ospf6' | 'anycast'> = []
  if (routing.bgp)       leftSlots.push('bgp')
  if (routing.ospf)      leftSlots.push('ospf')
  if (routing.ospf6)     leftSlots.push('ospf6')
  if (services.length)   leftSlots.push('anycast')

  const LEFT_HUB_X  = RX - 300
  const LEFT_LEAF_X = RX - 550
  const LEFT_NBR_X  = RX - 800
  const LEFT_SLOT_H = 280
  const leftHubY = (i: number) => RY + (i - (leftSlots.length - 1) / 2) * LEFT_SLOT_H

  const vrfEntries = Object.entries(routing.vrfs ?? {})
  const rightSlots: Array<'vrrp' | 'vrf'> = []
  if (vrrpInstances.length > 0) rightSlots.push('vrrp')
  if (vrfEntries.length > 0)    rightSlots.push('vrf')

  const RIGHT_HUB_X  = RX + 300
  const RIGHT_LEAF_X = RX + 540
  const RIGHT_SLOT_H = 300
  const rightHubY = (i: number) => RY + (i - (rightSlots.length - 1) / 2) * RIGHT_SLOT_H

  const bgpNeighbors = routing.bgp?.neighbors ?? []
  if (routing.bgp) {
    const bgpIdx = leftSlots.indexOf('bgp')
    const bgpPos = { x: LEFT_HUB_X, y: leftHubY(bgpIdx) }
    push({
      id: 'bgp',
      type: 'group',
      data: { label: `BGP AS${routing.bgp.asn}`, style: 'bgp', selected: sel('bgp'), nodeTitle: `BGP AS${routing.bgp.asn}` },
      position: bgpPos,
      style: { ...S.bgp, outline: sel('bgp') ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' },
    })
    connect('bgp:router', bgpPos, routerPos, { animated: true, style: { stroke: '#2563eb' }, label: 'BGP', labelStyle: { fill: '#2563eb', fontSize: 10 } })

    const nSpacing = 90
    const nStartY = bgpPos.y - ((bgpNeighbors.length - 1) * nSpacing) / 2
    bgpNeighbors.forEach((n, i) => {
      const id = `bgp-${n.address}`
      const peerPos = { x: LEFT_LEAF_X, y: nStartY + i * nSpacing }
      push({
        id,
        type: 'group',
        data: {
          label: (
            <div className="text-center leading-tight">
              <div className="font-bold font-mono text-[11px]">{n.address}</div>
              <div className="opacity-80 text-[10px]">AS{n.remote_asn}</div>
              {n.description && <div className="opacity-60 text-[10px]">{n.description}</div>}
            </div>
          ),
          style: 'bgp',
          selected: sel(id),
          nodeTitle: `BGP ${n.address} / AS${n.remote_asn}`,
        },
        position: peerPos,
        style: { ...S.bgp, outline: sel(id) ? '2px solid white' : 'none', outlineOffset: 2 },
      })
      connect(`bgp:${id}`, bgpPos, peerPos, { animated: true, style: { stroke: '#2563eb', opacity: 0.7 } })
    })
  }

  const areas = routing.ospf?.areas ?? []
  if (routing.ospf) {
    const ospfIdx = leftSlots.indexOf('ospf')
    const ospfPos = { x: LEFT_HUB_X, y: leftHubY(ospfIdx) }
    push({
      id: 'ospf',
      type: 'group',
      data: { label: 'OSPF', style: 'ospf', selected: sel('ospf'), nodeTitle: 'OSPF' },
      position: ospfPos,
      style: { ...S.ospf, outline: sel('ospf') ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' },
    })
    connect('ospf:router', ospfPos, routerPos, { label: 'OSPF', style: { stroke: '#9333ea' }, labelStyle: { fill: '#9333ea', fontSize: 10 } })

    const neighborsByArea: Record<string, OSPFNeighbor[]> = {}
    for (const n of ospfNeighbors) {
      const key = normalizeAreaId(n.area ?? '0.0.0.0')
      neighborsByArea[key] = [...(neighborsByArea[key] ?? []), n]
    }

    const normalizedAreas = areas.map((a) => ({ ...a, id: normalizeAreaId(a.id) }))
    const configAreaIds = new Set(normalizedAreas.map((a) => a.id))
    const allAreas: OSPFArea[] = [
      ...normalizedAreas,
      ...Object.keys(neighborsByArea)
        .filter((id) => !configAreaIds.has(id))
        .map((id) => ({ id, networks: [] as string[], type: undefined as string | undefined })),
    ]

    const areaSpacing = 110
    const areaStartY = ospfPos.y - ((allAreas.length - 1) * areaSpacing) / 2
    allAreas.forEach((a, aIdx) => {
      const areaId = `ospf-${a.id}`
      const areaPos = { x: LEFT_LEAF_X, y: areaStartY + aIdx * areaSpacing }
      push({
        id: areaId,
        type: 'group',
        data: {
          label: (
            <div className="text-center leading-tight">
              <div className="font-bold">Area {a.id}</div>
              {a.type && a.type !== 'normal' && <div className="opacity-80 text-[10px]">{a.type}</div>}
              {(a.networks ?? []).map((n) => (
                <div key={n} className="font-mono opacity-70 text-[10px]">{n}</div>
              ))}
            </div>
          ),
          style: 'ospf',
          selected: sel(areaId),
          nodeTitle: `OSPF Area ${a.id}`,
        },
        position: areaPos,
        style: { ...S.ospf, outline: sel(areaId) ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' },
      })
      connect(`${areaId}:ospf`, areaPos, ospfPos, { style: { stroke: '#9333ea', opacity: 0.7 } })

      const group = neighborsByArea[a.id] ?? []
      const nSpacing = 70
      const nStartY = areaPos.y - ((group.length - 1) * nSpacing) / 2
      group.forEach((n, nIdx) => {
        const nid = `ospfn-${n.neighbor_id}`
        const stateColor = OSPF_STATE_COLOR[n.state] ?? '#6b7280'
        const nbrPos = { x: LEFT_NBR_X, y: nStartY + nIdx * nSpacing }
        push({
          id: nid,
          type: 'group',
          data: {
            label: (
              <div className="text-center leading-tight">
                <div className="font-bold font-mono text-[11px]">{n.neighbor_id}</div>
                {n.interface && <div className="opacity-80 text-[10px]">{n.interface}</div>}
                <div className="text-[10px] font-semibold">{n.state}</div>
              </div>
            ),
            style: 'ospf',
            selected: sel(nid),
            nodeTitle: `OSPF neighbor ${n.neighbor_id}`,
          },
          position: nbrPos,
          style: { ...S.ospf, outline: sel(nid) ? '2px solid white' : 'none', outlineOffset: 2 },
        })
        connect(`${nid}:${areaId}`, nbrPos, areaPos, { animated: true, style: { stroke: stateColor } })
      })
    })
  }

  if (services.length > 0) {
    const anyIdx = leftSlots.indexOf('anycast')
    const anycastPos = { x: LEFT_HUB_X, y: leftHubY(anyIdx) }
    push({
      id: 'anycast',
      type: 'group',
      data: { label: 'Anycast', style: 'anycast', selected: sel('anycast'), nodeTitle: 'Anycast' },
      position: anycastPos,
      style: { ...S.anycast, outline: sel('anycast') ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' },
    })
    connect('anycast:router', anycastPos, routerPos, { label: 'anycast', style: { stroke: '#ea580c' }, labelStyle: { fill: '#ea580c', fontSize: 10 } })

    const svcSpacing = 95
    const svcStartY = anycastPos.y - ((services.length - 1) * svcSpacing) / 2
    services.forEach((svc, i) => {
      const id = `anycast-${i}`
      const svcPos = { x: LEFT_LEAF_X, y: svcStartY + i * svcSpacing }
      push({
        id,
        type: 'group',
        data: {
          label: (
            <div className="text-center leading-tight">
              <div className="font-bold">{svc.name}</div>
              <div className="opacity-80 text-[10px]">{svc.active ? '● active' : '○ inactive'}</div>
              {(svc.anycast_ips ?? []).map((ip) => (
                <div key={ip} className="font-mono opacity-70 text-[10px]">{ip}</div>
              ))}
            </div>
          ),
          style: 'anycast',
          selected: sel(id),
          nodeTitle: svc.name,
        },
        position: svcPos,
        style: { ...S.anycast, outline: sel(id) ? '2px solid white' : 'none', outlineOffset: 2 },
      })
      connect(`${id}:anycast`, svcPos, anycastPos, { style: { stroke: '#ea580c', opacity: 0.6 } })
    })
  }

  if (vrrpInstances.length > 0) {
    const vrrpIdx = rightSlots.indexOf('vrrp')
    const vrrpGroupPos = { x: RIGHT_HUB_X, y: rightHubY(vrrpIdx) }
    push({
      id: 'vrrp',
      type: 'group',
      data: { label: `VRRP (${vrrpInstances.length})`, style: 'vrrp', selected: sel('vrrp'), nodeTitle: 'VRRP' },
      position: vrrpGroupPos,
      style: { ...S.vrrp, outline: sel('vrrp') ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' },
    })
    connect('router:vrrp', routerPos, vrrpGroupPos, { label: 'VRRP', style: { stroke: '#0f766e' }, labelStyle: { fill: '#0f766e', fontSize: 10 } })

    const instSpacing = 120
    const instStartY = vrrpGroupPos.y - ((vrrpInstances.length - 1) * instSpacing) / 2
    vrrpInstances.forEach((inst, iIdx) => {
      const instKey = inst.name ?? `VI_${inst.interface}_${inst.id}`
      const instId = `vrrp-${inst.interface}-${inst.id}`
      const stateColor = VRRP_STATE_COLOR[inst.state] ?? '#6b7280'
      const instPos = { x: RIGHT_LEAF_X, y: instStartY + iIdx * instSpacing }
      push({
        id: instId,
        type: 'group',
        data: {
          label: (
            <div className="text-center leading-tight">
              <div className="font-bold text-[11px]">{instKey}</div>
              <div className="opacity-80 text-[10px]">{inst.state}</div>
              {(inst.vips ?? []).slice(0, 2).map((vip) => (
                <div key={vip} className="font-mono opacity-70 text-[10px]">{vip}</div>
              ))}
              {(inst.vips ?? []).length > 2 && <div className="opacity-60 text-[10px]">+{(inst.vips ?? []).length - 2} more</div>}
            </div>
          ),
          style: 'vrrp',
          selected: sel(instId),
          nodeTitle: `${instKey} · ${inst.state}`,
        },
        position: instPos,
        style: { background: stateColor, color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 11, width: 'max-content', outline: sel(instId) ? '2px solid white' : 'none', outlineOffset: 2 },
      })
      connect(`${instId}:vrrp`, instPos, vrrpGroupPos, { style: { stroke: stateColor, opacity: 0.7 } })

      const peers = inst.peers ?? []
      const peerSpacing = 65
      const peerStartY = instPos.y - ((peers.length - 1) * peerSpacing) / 2
      peers.forEach((peer, pIdx) => {
        const peerId = `vrrp-peer-${inst.interface}-${inst.id}-${peer.ip}`
        const peerPos = { x: RIGHT_LEAF_X + 240, y: peerStartY + pIdx * peerSpacing }
        push({
          id: peerId,
          type: 'group',
          data: {
            label: (
              <div className="text-center leading-tight">
                <div className="font-bold font-mono text-[11px]">{peer.ip}</div>
                <div className="opacity-80 text-[10px]">prio {peer.priority}</div>
                {peer.last_seen && <div className="opacity-60 text-[10px]">{peer.last_seen}</div>}
              </div>
            ),
            style: 'vrrp',
            nodeTitle: `VRRP peer ${peer.ip}`,
          },
          position: peerPos,
          style: S.vrrp,
        })
        connect(`${peerId}:${instId}`, peerPos, instPos, { animated: true, style: { stroke: '#0f766e', opacity: 0.7 } })
      })
    })
  }

  if (routing.ospf6) {
    const ospf6Idx = leftSlots.indexOf('ospf6')
    const ospf6Pos = { x: LEFT_HUB_X, y: leftHubY(ospf6Idx) }
    push({
      id: 'ospf6',
      type: 'group',
      data: { label: 'OSPFv3', style: 'ospf6', selected: sel('ospf6'), nodeTitle: 'OSPFv3' },
      position: ospf6Pos,
      style: { ...S.ospf6, outline: sel('ospf6') ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' },
    })
    connect('ospf6:router', ospf6Pos, routerPos, { label: 'OSPFv3', style: { stroke: '#0d9488' }, labelStyle: { fill: '#0d9488', fontSize: 10 } })

    const ospf6Areas = routing.ospf6.areas ?? []
    const areaSpacing = 110
    const areaStartY = ospf6Pos.y - ((ospf6Areas.length - 1) * areaSpacing) / 2
    ospf6Areas.forEach((a, aIdx) => {
      const areaId = `ospf6-area-${a.id}`
      const areaPos = { x: LEFT_LEAF_X, y: areaStartY + aIdx * areaSpacing }
      push({
        id: areaId,
        type: 'group',
        data: {
          label: (
            <div className="text-center leading-tight">
              <div className="font-bold">Area {normalizeAreaId(a.id)}</div>
              {a.type && a.type !== 'normal' && <div className="opacity-80 text-[10px]">{a.type}</div>}
              {(a.ranges ?? []).map((r) => (
                <div key={r} className="font-mono opacity-70 text-[10px]">{r}</div>
              ))}
            </div>
          ),
          style: 'ospf6',
          selected: sel(areaId),
          nodeTitle: `OSPFv3 Area ${normalizeAreaId(a.id)}`,
        },
        position: areaPos,
        style: { ...S.ospf6, outline: sel(areaId) ? '2px solid white' : 'none', outlineOffset: 2 },
      })
      connect(`${areaId}:ospf6`, areaPos, ospf6Pos, { style: { stroke: '#0d9488', opacity: 0.7 } })
    })
  }

  if (vrfEntries.length > 0) {
    const vrfIdx = rightSlots.indexOf('vrf')
    const vrfHubPos = { x: RIGHT_HUB_X, y: rightHubY(vrfIdx) }
    push({
      id: 'vrf',
      type: 'group',
      data: { label: `VRFs (${vrfEntries.length})`, style: 'vrf', selected: sel('vrf'), nodeTitle: 'VRFs' },
      position: vrfHubPos,
      style: { ...S.vrf, outline: sel('vrf') ? '2px solid white' : 'none', outlineOffset: 2, cursor: 'pointer' },
    })
    connect('router:vrf', routerPos, vrfHubPos, { label: 'VRF', style: { stroke: '#b45309' }, labelStyle: { fill: '#b45309', fontSize: 10 } })

    const vrfSpacing = 110
    const vrfStartY = vrfHubPos.y - ((vrfEntries.length - 1) * vrfSpacing) / 2
    vrfEntries.forEach(([name, vr], i) => {
      const vrfNodeId = `vrf-${name}`
      const vrfPos = { x: RIGHT_LEAF_X, y: vrfStartY + i * vrfSpacing }
      const decl = vrfDeclarations[name]
      const protos: string[] = []
      if (vr.bgp) protos.push(`BGP ${vr.bgp.asn ?? ''}`.trim())
      if (vr.ospf) protos.push('OSPF')
      if (vr.ospf6) protos.push('OSPFv3')
      if (vr.static?.length) protos.push(`${vr.static.length} static`)
      push({
        id: vrfNodeId,
        type: 'group',
        data: {
          label: (
            <div className="text-center leading-tight">
              <div className="font-bold">{name}</div>
              {decl && <div className="opacity-70 text-[10px]">table {decl.table}</div>}
              {protos.map((p) => <div key={p} className="opacity-80 text-[10px]">{p}</div>)}
            </div>
          ),
          style: 'vrf',
          selected: sel(vrfNodeId),
          nodeTitle: `VRF ${name}`,
        },
        position: vrfPos,
        style: { ...S.vrf, outline: sel(vrfNodeId) ? '2px solid white' : 'none', outlineOffset: 2 },
      })
      connect(`${vrfNodeId}:vrf`, vrfPos, vrfHubPos, { style: { stroke: '#b45309', opacity: 0.7 } })
    })
  }

  const tunnelEntries = Object.entries(tunnels ?? {}).slice(0, 6)
  const wgEntries     = Object.entries(wireguard ?? {}).slice(0, 6)
  const bottomItems = [
    ...tunnelEntries.map(([n, t]) => ({
      id: `tunnel-${n}`, styleKey: 'tunnel', edgeColor: '#0891b2', edgeLabel: t.mode,
      nodeTitle: `Tunnel ${n}`,
      label: (
        <div className="text-center leading-tight">
          <div className="font-bold">{n}</div>
          <div className="opacity-80 text-[10px] font-mono">{t.mode}</div>
          {t.local && t.remote && <div className="opacity-60 text-[10px] font-mono">{t.local} → {t.remote}</div>}
        </div>
      ),
    })),
    ...wgEntries.map(([n, w]) => ({
      id: `wg-${n}`, styleKey: 'wireguard', edgeColor: '#4f46e5', edgeLabel: 'WireGuard',
      nodeTitle: `WireGuard ${n}`,
      label: (
        <div className="text-center leading-tight">
          <div className="font-bold">{n}</div>
          {w.listen_port && <div className="opacity-80 text-[10px]">:{w.listen_port}</div>}
          <div className="opacity-60 text-[10px]">{(w.peers ?? []).length} peers</div>
        </div>
      ),
    })),
  ]
  if (bottomItems.length > 0) {
    const spacing = 220
    const startX  = RX - ((bottomItems.length - 1) * spacing) / 2
    bottomItems.forEach((item, i) => {
      const childPos = { x: startX + i * spacing, y: RY + 260 }
      push({
        id: item.id,
        type: 'group',
        data: { label: item.label, style: item.styleKey, nodeTitle: item.nodeTitle },
        position: childPos,
        style: S[item.styleKey],
      })
      connect(`router:${item.id}`, routerPos, childPos, { label: item.edgeLabel, style: { stroke: item.edgeColor }, labelStyle: { fill: item.edgeColor, fontSize: 10 } })
    })
  }

  return { nodes, edges }
}

export const PROTO_FILTER: Record<string, string[]> = {
  all:     [],
  bgp:     ['bgp', 'zebra'],
  ospf:    ['ospf'],
  ospf6:   ['ospf6'],
  static:  ['routier', 'static'],
  anycast: [],
  vrf:     [],
}

export function protoForNode(id: string): string | null {
  if (id === 'router')                                                          return 'all'
  if (id === 'bgp'     || id.startsWith('bgp-'))                              return 'bgp'
  if (id === 'ospf6'   || id.startsWith('ospf6-'))                            return 'ospf6'
  if (id === 'ospf'    || id.startsWith('ospf-') || id.startsWith('ospfn-')) return 'ospf'
  if (id === 'static'  || id.startsWith('route-'))                            return 'static'
  if (id === 'anycast' || id.startsWith('anycast-'))                          return 'anycast'
  if (id === 'vrrp'    || id.startsWith('vrrp-'))                             return 'vrrp'
  if (id === 'vrf'     || id.startsWith('vrf-'))                              return 'vrf'
  return null
}

