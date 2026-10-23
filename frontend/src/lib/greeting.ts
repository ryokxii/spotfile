// The Dashboard greeting and the sidebar initials, from the display name.

export type PartOfDay = 'morning' | 'afternoon' | 'evening'

// 5:00–11:59 is morning, 12:00–17:59 afternoon, the rest evening.
export function partOfDay(hour: number): PartOfDay {
  if (hour >= 5 && hour < 12) return 'morning'
  if (hour >= 12 && hour < 18) return 'afternoon'
  return 'evening'
}

const words = (name: string): string[] => name.trim().split(/\s+/).filter(Boolean)

// "Good evening, Zafer" — first name only; "Good evening" without a name.
export function greeting(hour: number, name: string): string {
  const salutation = `Good ${partOfDay(hour)}`
  const firstName = words(name)[0]
  return firstName ? `${salutation}, ${firstName}` : salutation
}

const firstLetter = (word: string): string => (Array.from(word)[0] ?? '').toUpperCase()

// "ZA" for "Zafer Acar", "Z" for "Zafer", "" without a name.
export function initials(name: string): string {
  const parts = words(name)
  if (parts.length === 0) return ''
  if (parts.length === 1) return firstLetter(parts[0])
  return firstLetter(parts[0]) + firstLetter(parts[parts.length - 1])
}
