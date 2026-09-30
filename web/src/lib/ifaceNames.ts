import { useMemo } from 'react'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'

interface IfaceCfg {
  select?: string
  type?: string
}
interface CfgShape {
  interfaces?: Record<string, IfaceCfg>
}

const isLiteralDevice = (sel: string) =>
  !/^[a-zA-Z]+\[\d+\]$/.test(sel) && !/^mac\(/.test(sel)

export function buildIfaceLabelMap(cfg?: CfgShape | null): Record<string, string> {
  const map: Record<string, string> = {}
  for (const [name, iface] of Object.entries(cfg?.interfaces ?? {})) {
    if (iface?.type && iface.type !== 'physical') continue
    const sel = iface?.select
    if (!sel || !isLiteralDevice(sel)) continue
    map[sel] = name
  }
  return map
}

export function useIfaceLabels(): (device: string) => string {
  const { data: cfg } = useFetch<CfgShape>(() => api.apiConfigGet() as unknown as Promise<CfgShape>)
  const map = useMemo(() => buildIfaceLabelMap(cfg), [cfg])
  return (device: string) => {
    const name = map[device]
    return name && name !== device ? `${name} (${device})` : device
  }
}
