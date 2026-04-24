# Frontend guidelines

**Stack**: Vue 3 · Reka UI v2 · Tailwind CSS v4 · Vite

## Tooling
- **Vite** is the only build tool — no webpack, no CRA, no separate PostCSS config.
- Install Tailwind via the Vite plugin (`@tailwindcss/vite`), not PostCSS:
  ```ts
  // vite.config.ts
  import tailwindcss from '@tailwindcss/vite'
  export default defineConfig({ plugins: [tailwindcss()] })
  ```
- Import Tailwind in the root CSS file with a single line — no `tailwind.config.js` needed:
  ```css
  @import "tailwindcss";
  ```
- Keep `package.json` lean: the only required runtime dependencies are `vue`, `reka-ui`; dev dependencies are `vite`, `@tailwindcss/vite`, `tailwindcss`, `@vitejs/plugin-vue`, and TypeScript tooling.

## Vue 3
- Always use `<script setup lang="ts">` — never the Options API or `setup()` function form.
- Use `ref` for primitives, `reactive` for objects; prefer `ref` when in doubt (consistent `.value` access).
- Extract reusable logic into composables (`use*.ts`), not mixins or utilities.
- Group template, script, and style in a single `.vue` SFC; keep components small and focused.
- Use `defineProps`, `defineEmits`, `defineExpose` with TypeScript types directly — no runtime prop validation objects.

## Reka UI
- Use Reka UI headless components as the base for all interactive elements (dialogs, dropdowns, tooltips, etc.) — do not build custom accessible widgets from scratch.
- Style Reka parts exclusively with Tailwind utility classes on the component's `class` attribute — no scoped CSS for Reka elements.
- Compose Reka primitive parts directly in the template; only wrap them in a component when the same composition is reused in three or more places.

## Tailwind CSS v4
- Configuration is CSS-first: use `@theme`, `@layer`, and `@utility` in the root CSS file instead of a JS config.
- Use Tailwind utility classes directly in templates; avoid arbitrary values (`[...]`) unless there is no standard scale equivalent.
- Do not use `@apply` — compose utilities in the template, or extract a component if reuse is needed.
