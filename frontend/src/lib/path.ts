// Filesystem path helpers. Paths may use either separator (macOS/Linux "/",
// Windows "\"), so split on both.

const SEPARATORS = /[/\\]/

// The final path segment, e.g. "/a/b/notes.md" -> "notes.md".
export function filename(path: string): string {
  return path.split(SEPARATORS).pop() || path
}

// The directory portion, e.g. "/a/b/notes.md" -> "/a/b".
export function dirpath(path: string): string {
  const parts = path.split(SEPARATORS)
  parts.pop()
  return parts.join('/') || path
}
