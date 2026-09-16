// WCAG 2.x contrast ratio between two opaque sRGB colours.
// https://www.w3.org/TR/WCAG21/#dfn-contrast-ratio

export interface RGB {
  r: number
  g: number
  b: number
}

const HEX = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i

export function parseHex(hex: string): RGB {
  const m = HEX.exec(hex.trim())
  if (!m) throw new Error(`expected #RRGGBB, got "${hex}"`)
  return { r: parseInt(m[1], 16), g: parseInt(m[2], 16), b: parseInt(m[3], 16) }
}

function channel(value: number): number {
  const c = value / 255
  return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

function luminance({ r, g, b }: RGB): number {
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

export function contrastRatio(a: string, b: string): number {
  const [hi, lo] = [luminance(parseHex(a)), luminance(parseHex(b))].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}
