// Decode a base64 payload (from Go's ReadFileAsBase64) as UTF-8 text.
//
// atob() returns a Latin-1 "binary string" — one char per byte — so any
// multi-byte UTF-8 character (en dash "–", smart quotes, emoji, accents)
// mojibakes when shown directly (e.g. "1–10" renders as "1â10"). Rebuild the
// byte array and decode it as UTF-8 instead.
export function decodeBase64Utf8(b64: string): string {
  const binary = atob(b64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return new TextDecoder('utf-8').decode(bytes)
}
