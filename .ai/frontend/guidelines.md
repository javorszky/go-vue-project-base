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

## Security headers

CSP must be set wherever `index.html` is served — the static host/CDN in production, and Vite's dev server in development. It is not the backend's responsibility (the backend serves JSON, not HTML pages).

### Policy directives for this SPA
```
Content-Security-Policy:
  default-src 'self';
  script-src  'self';
  style-src   'self' 'unsafe-inline';
  connect-src 'self' https://api.example.com;
  img-src     'self' data:;
  font-src    'self';
  object-src  'none';
  base-uri    'self';
  form-action 'self';
  frame-ancestors 'none';
```

Notes:
- `unsafe-inline` on `style-src` is required in **development** (Vite injects styles at runtime). In a production build Vite extracts all styles to `.css` files, so you can drop `unsafe-inline` there.
- `connect-src` must include the backend API origin. In development that is `http://localhost:8080`; in production it is the deployed API origin.
- Replace `https://api.example.com` with the real production API origin before deploying.

### Vite dev server configuration
Set headers in `vite.config.ts` so the dev environment matches production as closely as possible:
```ts
server: {
  headers: {
    'Content-Security-Policy':
      "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
      "connect-src 'self' http://localhost:8080; img-src 'self' data:; " +
      "font-src 'self'; object-src 'none'; base-uri 'self'; " +
      "form-action 'self'; frame-ancestors 'none';",
  },
}
```

### Production (static host / CDN)
Configure the CSP header at the hosting layer (e.g. Nginx `add_header`, Cloudflare Transform Rules, Netlify `_headers` file) — not in the built HTML file itself. Drop `unsafe-inline` from `style-src` in the production policy.

## Tailwind CSS v4
- Configuration is CSS-first: use `@theme`, `@layer`, and `@utility` in the root CSS file instead of a JS config.
- Use Tailwind utility classes directly in templates; avoid arbitrary values (`[...]`) unless there is no standard scale equivalent.
- Do not use `@apply` — compose utilities in the template, or extract a component if reuse is needed.
