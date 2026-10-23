// Short human times for the Dashboard: "3 min ago", "yesterday", "14 months".

const MINUTE = 60_000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

// When something happened, relative to now.
export function ago(when: Date, now: Date): string {
  const ms = now.getTime() - when.getTime()
  if (ms < MINUTE) return 'just now'
  if (ms < HOUR) return `${Math.floor(ms / MINUTE)} min ago`
  if (ms < DAY) return `${Math.floor(ms / HOUR)}h ago`
  if (ms < 2 * DAY) return 'yesterday'
  if (ms < 7 * DAY) return WEEKDAYS[when.getDay()]
  return `${MONTHS[when.getMonth()]} ${when.getDate()}`
}

// How long something has gone unused: days under 60, then whole months.
export function age(since: Date, now: Date): string {
  const days = Math.floor((now.getTime() - since.getTime()) / DAY)
  if (days < 60) return `${days} ${days === 1 ? 'day' : 'days'}`
  return `${Math.floor(days / 30)} months`
}
