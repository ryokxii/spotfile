import { get, writable } from 'svelte/store'
import { describe, expect, it } from 'vitest'
import { createMotionStore } from './motion'
import type { MotionPreference } from './settings'

describe('createMotionStore', () => {
  it('stamps data-motion for reduce and full, and removes it for system', () => {
    const root = document.createElement('html')
    const preference = writable<MotionPreference>('reduce')
    const motion = createMotionStore(root, preference)
    const stop = motion.subscribe(() => undefined)
    expect(root.dataset.motion).toBe('reduce')

    preference.set('full')
    expect(root.dataset.motion).toBe('full')

    preference.set('system')
    expect(root.hasAttribute('data-motion')).toBe(false)
    expect(get(motion)).toBe('system')
    stop()
  })
})
