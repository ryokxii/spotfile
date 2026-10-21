// Applies the reduce-motion preference as data-motion on the root element.
// tokens.css zeroes the motion tokens for "reduce", and for the OS setting
// unless the preference is "full".

import { derived, type Readable } from 'svelte/store'
import type { MotionPreference } from './settings'

export function createMotionStore(root: HTMLElement, preference: Readable<MotionPreference>): Readable<MotionPreference> {
  return derived(preference, (pref) => {
    if (pref === 'system') delete root.dataset.motion
    else root.dataset.motion = pref
    return pref
  })
}
