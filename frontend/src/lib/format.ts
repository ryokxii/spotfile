// Display formatting for figures and paths.

const STEP = 1024
const UNITS = ['KB', 'MB', 'GB']
const ELLIPSIS = '…'

// Bytes as a short size: "512 B", "2.1 MB". One decimal from KB up.
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < STEP) return `${Math.max(0, Math.round(bytes) || 0)} B`
  let value = bytes / STEP
  let unit = 0
  while (value >= STEP && unit < UNITS.length - 1) {
    value /= STEP
    unit++
  }
  return `${value.toFixed(1)} ${UNITS[unit]}`
}

// Shortens path to at most max characters by cutting from the middle, keeping
// the last segment (the folder's own name) whole when it fits.
export function truncateMiddle(path: string, max: number): string {
  if (path.length <= max) return path
  const cut = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
  const tail = cut >= 0 ? path.slice(cut) : ''
  const room = max - tail.length - ELLIPSIS.length
  if (tail && room > 0) return path.slice(0, room) + ELLIPSIS + tail

  const keep = max - ELLIPSIS.length
  const head = Math.ceil(keep / 2)
  return path.slice(0, head) + ELLIPSIS + path.slice(path.length - (keep - head))
}
