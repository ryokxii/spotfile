// Base64 decoding helpers for payloads from Go's ReadFileAsBase64.

// Decode a base64 string to raw bytes. atob() returns a Latin-1 "binary
// string" (one char per byte); this rebuilds the actual byte array.
export function base64ToBytes(b64: string): Uint8Array {
  const binary = atob(b64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}

// Decode a base64 payload as UTF-8 text. Going through the byte array avoids
// the mojibake atob() alone produces for multi-byte characters (en dash "–",
// smart quotes, accents, emoji) — e.g. "1–10" rendering as "1â10".
export function decodeBase64Utf8(b64: string): string {
  return new TextDecoder('utf-8').decode(base64ToBytes(b64))
}
