import './styles/fonts.css'
import './styles/tokens.css'
import './styles/global.css'
import App from './App.svelte'
import { settings } from './stores/app'

// Load saved appearance before mounting so the first frame uses the right
// theme and motion. load() never rejects; it falls back to defaults.
void settings.load().then(() => {
  new App({ target: document.getElementById('app') as HTMLElement })
})
