import './styles/fonts.css'
import './styles/tokens.css'
import './styles/global.css'
import App from './App.svelte'
import { profile, settings } from './stores/app'

// Load saved appearance and profile before mounting so the first frame uses the
// right theme, motion and name. load() never rejects; it falls back to defaults.
void Promise.all([settings.load(), profile.load()]).then(() => {
  new App({ target: document.getElementById('app') as HTMLElement })
})
