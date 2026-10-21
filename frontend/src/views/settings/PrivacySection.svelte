<script lang="ts">
  // Privacy: whether history is kept, when files count as stale, clearing history.
  import { profile } from '../../stores/app'
  import { STALE_DAY_CHOICES, type StaleDays } from '../../stores/profile'
  import Button from '../../ui/Button.svelte'
  import ConfirmDialog from '../../ui/ConfirmDialog.svelte'
  import SegmentedControl from '../../ui/SegmentedControl.svelte'
  import SettingRow from './SettingRow.svelte'
  import SettingsSection from './SettingsSection.svelte'

  const HISTORY = [
    { value: 'on', label: 'On' },
    { value: 'off', label: 'Off' },
  ]
  const STALE = STALE_DAY_CHOICES.map((d) => ({ value: String(d), label: `${d} days` }))

  let confirming = false

  const onHistory = (e: CustomEvent<string>) => profile.setHistory(e.detail === 'on')
  const onStale = (e: CustomEvent<string>) => profile.setStaleDays(Number(e.detail) as StaleDays)

  async function clear() {
    confirming = false
    await profile.clearHistory()
  }
</script>

<SettingsSection title="Privacy">
  <SettingRow label="Remember searches and opened files" hint="Shown on the Dashboard. Stored only on this computer in ~/.spotfile">
    <SegmentedControl label="Remember searches and opened files" options={HISTORY} value={$profile.history ? 'on' : 'off'} on:change={onHistory} />
  </SettingRow>
  <SettingRow label="Stale after" hint="Files not changed or opened for this long appear in Free up space">
    <SegmentedControl label="Stale after" options={STALE} value={String($profile.staleDays)} on:change={onStale} />
  </SettingRow>
  <SettingRow label="Clear history" hint="Forgets recent searches and opened files">
    <Button variant="destructive" on:click={() => (confirming = true)}>Clear history…</Button>
  </SettingRow>
  {#if $profile.privacyError}
    <p class="error" role="alert">{$profile.privacyError}</p>
  {/if}
</SettingsSection>

<ConfirmDialog open={confirming} title="Clear your history?" confirmLabel="Clear history" on:confirm={clear} on:cancel={() => (confirming = false)}>
  <p>Recent searches and opened files are forgotten. Your files and the index stay as they are.</p>
</ConfirmDialog>

<style>
  .error {
    font-size: var(--text-small-size);
    color: var(--color-urgent);
  }
</style>
