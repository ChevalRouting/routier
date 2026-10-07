import type { IpcalcSubnetInfo } from '@/api'
import { CalcRow, errorMessage, ErrorNote, ResultTable, ToolInput, useDebounced } from '@/components/ip-tools/shared'
import { api } from '@/lib/client'
import { Badge } from 'cheval-ui'
import { useEffect, useState } from 'react'

export function SubnetTool() {
  const [input, setInput] = useState('100.64.12.192/27')
  const query = useDebounced(input.trim())
  const [info, setInfo] = useState<IpcalcSubnetInfo | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!query) {
      setInfo(null)
      setError(null)
      return
    }
    let live = true
    api.apiToolsSubnetGet({ cidr: query })
      .then((res) => { if (live) { setInfo(res as IpcalcSubnetInfo); setError(null) } })
      .catch((err) => { if (live) { setInfo(null); setError(errorMessage(err)) } })
    return () => { live = false }
  }, [query])

  const is4 = info?.family === 'v4'
  const rows: CalcRow[] = info
    ? [
        { label: 'Address', value: info.address, bits: info.address_bits },
        { label: 'Netmask', value: is4 ? `${info.netmask} = ${info.prefix}` : info.netmask, bits: info.netmask_bits },
        { label: 'Wildcard', value: info.wildcard, bits: info.wildcard_bits },
        { label: info.host_route ? 'Hostroute' : 'Network', value: info.network, bits: info.network_bits },
        ...(info.host_route ? [] : [
          { label: 'HostMin', value: info.host_min, bits: info.host_min_bits },
          { label: 'HostMax', value: info.host_max, bits: info.host_max_bits },
        ]),
        ...(info.broadcast ? [{ label: 'Broadcast', value: info.broadcast, bits: info.broadcast_bits }] : []),
      ]
    : []

  return (
    <div className="space-y-4">
      <ToolInput
        label="Address or CIDR"
        hint="e.g. 100.64.12.192/27 or 2001:db8::/64. A bare address is treated as a host route."
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder="100.64.12.192/27"
      />
      {error && <ErrorNote message={error} />}
      {info && (
        <>
          <ResultTable rows={rows} prefix={info.prefix} is4={is4} />
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary">Hosts/Net: {info.hosts}</Badge>
            {info._class && <Badge variant="outline">Class {info._class}</Badge>}
            <Badge variant="outline">{info.scope}</Badge>
            <Badge variant="outline">{info.family === 'v4' ? 'IPv4' : 'IPv6'}</Badge>
          </div>
        </>
      )}
    </div>
  )
}
