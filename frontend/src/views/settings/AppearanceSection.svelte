<script lang="ts">
  import { settings } from '../../stores/app'
  import type { MotionPreference, ThemePreference } from '../../stores/settings'
  import SegmentedControl from '../../ui/SegmentedControl.svelte'
  import SettingRow from './SettingRow.svelte'
  import SettingsSection from './SettingsSection.svelte'

  const THEMES = [
    { value: 'system', label: 'System' },
    { value: 'nocturne', label: 'Nocturne' },
    { value: 'daylight', label: 'Daylight' },
  ]
  const MOTION = [
    { value: 'system', label: 'System' },
    { value: 'reduce', label: 'On' },
    { value: 'full', label: 'Off' },
  ]

  const onTheme = (e: CustomEvent<string>) => settings.setTheme(e.detail as ThemePreference)
  const onMotion = (e: CustomEvent<string>) => settings.setMotion(e.detail as MotionPreference)
</script>

<SettingsSection title="Appearance">
  <SettingRow label="Theme" hint="System uses Nocturne when your computer is in dark mode, Daylight when it's light">
    <SegmentedControl label="Theme" options={THEMES} value={$settings.theme} on:change={onTheme} />
  </SettingRow>
  <SettingRow label="Reduce motion" hint="Turns off slides and fades. System follows your accessibility setting">
    <SegmentedControl label="Reduce motion" options={MOTION} value={$settings.motion} on:change={onMotion} />
  </SettingRow>
  {#if $settings.error}
    <p class="error" role="alert">{$settings.error}</p>
  {/if}
</SettingsSection>

<style>
  .error {
    font-size: var(--text-small-size);
    color: var(--color-urgent);
  }
</style>
