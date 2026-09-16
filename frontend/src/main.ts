import './styles/fonts.css'
import './styles/tokens.css'
import './styles/global.css'
import App from './App.svelte'

const app = new App({
  target: document.getElementById('app') as HTMLElement,
})

export default app
