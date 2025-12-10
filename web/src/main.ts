// web/src/main.ts
// single-spa Vue 3 entry point with lifecycle methods
import { h, createApp, App as VueApp } from 'vue'
import { createPinia } from 'pinia'
import singleSpaVue from 'single-spa-vue'
import App from './App.vue'
import router from './router'

// Type for single-spa custom props
interface CustomProps {
  domElement?: HTMLElement
  name?: string
  [key: string]: unknown
}

// Create the Vue lifecycle wrapper for single-spa
const vueLifecycles = singleSpaVue({
  createApp,
  appOptions: {
    render() {
      return h(App, {
        // Props passed from shell application
        ...(this as unknown as { customProps: CustomProps }).customProps,
      })
    },
  },
  handleInstance: (app: VueApp) => {
    app.use(createPinia())
    app.use(router)
  },
})

// Export single-spa lifecycle methods
export const bootstrap = vueLifecycles.bootstrap
export const mount = vueLifecycles.mount
export const unmount = vueLifecycles.unmount

// For standalone development (not in single-spa shell)
declare global {
  interface Window {
    singleSpaNavigate?: unknown
  }
}

if (!window.singleSpaNavigate) {
  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.mount('#app')
}

