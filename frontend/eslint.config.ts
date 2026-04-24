import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'
import pluginA11y from 'eslint-plugin-vuejs-accessibility'
import eslintConfigPrettier from 'eslint-config-prettier'

export default tseslint.config(
  // Exclude build output and dependencies
  { ignores: ['dist/**', 'node_modules/**'] },

  // TypeScript type-aware rules for .ts files
  {
    files: ['**/*.ts'],
    extends: [...tseslint.configs.recommendedTypeChecked],
    languageOptions: {
      parserOptions: {
        project: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
  },

  // Vue + TypeScript type-aware rules for .vue files.
  // vue-eslint-parser is the outer parser (handles the SFC structure);
  // tseslint.parser is the inner parser (handles <script lang="ts"> blocks).
  {
    files: ['**/*.vue'],
    extends: [
      ...tseslint.configs.recommendedTypeChecked,
      ...pluginVue.configs['flat/recommended'],
      ...pluginA11y.configs['flat/recommended'],
    ],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        project: true,
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
      // Enforce <script setup lang="ts"> — no Options API, no setup() function form
      'vue/component-api-style': ['error', ['script-setup']],

      // Semantic self-closing:
      //   void elements (br, img, input) → always self-close
      //   normal HTML elements (div, span) → never self-close
      //   Vue components → always self-close
      'vue/html-self-closing': [
        'error',
        { html: { void: 'always', normal: 'never', component: 'always' } },
      ],

      // Disallow deprecated slot syntax
      'vue/no-deprecated-slot-attribute': 'error',
      'vue/no-deprecated-slot-scope-attribute': 'error',

      // Require lang="ts" on all script blocks
      'vue/block-lang': ['error', { script: { lang: 'ts' } }],
    },
  },

  // Prettier must be last — disables ESLint rules that would conflict with formatting
  eslintConfigPrettier,
)
