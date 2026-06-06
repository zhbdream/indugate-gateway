/// <reference types="vite/client" />

declare module 'axios' {
  export interface AxiosRequestConfig {
    /** When true, suppress global error toast for this request. */
    silent?: boolean
  }
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
