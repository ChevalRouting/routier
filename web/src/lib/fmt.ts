export function fmtBytes(b: number): string {
  if (b >= 1e9) return (b / 1e9).toFixed(1) + ' GB'
  if (b >= 1e6) return (b / 1e6).toFixed(1) + ' MB'
  if (b >= 1e3) return (b / 1e3).toFixed(1) + ' KB'
  return b + ' B'
}

export function fmtBitrate(Bps: number): string {
  const bps = Bps * 8
  if (bps >= 1e9) return (bps / 1e9).toFixed(2) + ' Gbps'
  if (bps >= 1e6) return (bps / 1e6).toFixed(1) + ' Mbps'
  if (bps >= 1e3) return (bps / 1e3).toFixed(1) + ' Kbps'
  return bps.toFixed(0) + ' bps'
}

export function fmtPps(pps: number): string {
  if (pps >= 1e6) return (pps / 1e6).toFixed(1) + 'M pps'
  if (pps >= 1e3) return (pps / 1e3).toFixed(1) + 'k pps'
  return pps.toFixed(0) + ' pps'
}

export function fmtRate(r: number): string {
  if (r >= 1e6) return (r / 1e6).toFixed(1) + 'M/s'
  if (r >= 1e3) return (r / 1e3).toFixed(1) + 'k/s'
  return r.toFixed(0) + '/s'
}

export function fmtUptime(s: number): string {
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  const parts: string[] = []
  if (d) parts.push(`${d}d`)
  if (h) parts.push(`${h}h`)
  parts.push(`${m}m`)
  return parts.join(' ')
}
