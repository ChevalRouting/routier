const liveSkipKey = 'routier_live_skip'

export function formatSize(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1000 && unit < units.length - 1) {
    value /= 1000
    unit += 1
  }
  return `${value >= 100 || unit === 0 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`
}

export function liveSkipped(): boolean {
  try {
    return sessionStorage.getItem(liveSkipKey) === '1'
  } catch {
    return false
  }
}

export function skipLive(): void {
  try {
    sessionStorage.setItem(liveSkipKey, '1')
  } catch {
    return
  }
}
