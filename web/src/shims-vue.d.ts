/* eslint-disable */
declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare module 'single-spa-vue' {
  import type { App } from 'vue'
  
  interface SingleSpaVueOpts {
    createApp: typeof import('vue').createApp
    appOptions: {
      render: () => ReturnType<typeof import('vue').h>
    }
    handleInstance?: (app: App) => void
  }
  
  interface SingleSpaVueLifecycles {
    bootstrap: (props: Record<string, unknown>) => Promise<void>
    mount: (props: Record<string, unknown>) => Promise<void>
    unmount: (props: Record<string, unknown>) => Promise<void>
  }
  
  export default function singleSpaVue(opts: SingleSpaVueOpts): SingleSpaVueLifecycles
}

