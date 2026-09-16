// Stroke icons on a 20×20 grid, drawn in currentColor.

export type IconName = 'search' | 'folder' | 'file' | 'close' | 'chevron-left' | 'chevron-right' | 'minus' | 'plus' | 'target'

export const ICONS: Record<IconName, string[]> = {
  search: ['M8.5 14a5.5 5.5 0 1 0 0-11 5.5 5.5 0 0 0 0 11Z', 'M13 13l3.5 3.5'],
  folder: ['M2.5 6a1.5 1.5 0 0 1 1.5-1.5h3.6l1.8 1.8H16A1.5 1.5 0 0 1 17.5 7.8v6.7A1.5 1.5 0 0 1 16 16H4a1.5 1.5 0 0 1-1.5-1.5V6Z'],
  file: ['M11.5 2.5H6A1.5 1.5 0 0 0 4.5 4v12A1.5 1.5 0 0 0 6 17.5h8a1.5 1.5 0 0 0 1.5-1.5V6.5l-4-4Z', 'M11.5 2.5v4h4'],
  close: ['M5 5l10 10', 'M15 5L5 15'],
  'chevron-left': ['M12 4.5L6.5 10l5.5 5.5'],
  'chevron-right': ['M8 4.5l5.5 5.5L8 15.5'],
  minus: ['M5 10h10'],
  plus: ['M10 5v10', 'M5 10h10'],
  target: ['M10 16.5a6.5 6.5 0 1 0 0-13 6.5 6.5 0 0 0 0 13Z', 'M10 12.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z'],
}
