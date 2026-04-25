/// <reference types="vite/client" />

// Declare project-specific env vars so they are typed as string rather than any.
// Add one entry per VITE_* variable used in the codebase.
interface ImportMetaEnv {
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// Gives TypeScript a concrete type for .vue imports when used outside vue-tsc
// (e.g. in typescript-eslint's type-aware analysis).
declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}
