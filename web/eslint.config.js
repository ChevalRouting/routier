import routier from './lint/rules.mjs'
import stylistic from '@stylistic/eslint-plugin'
import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'

export default tseslint.config(
  { ignores: ['dist', 'node_modules', 'src/api'] },
  {
    files: ['src/**/*.{ts,tsx}'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
    },
    linterOptions: { reportUnusedDisableDirectives: 'error' },
    plugins: {
      routier,
      stylistic,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      'routier/no-comments': 'error',
      'routier/shared-controls': 'error',
      'routier/named-handlers': 'error',
      'routier/named-types': 'error',
      'stylistic/no-trailing-spaces': 'error',
      'stylistic/eol-last': ['error', 'always'],
      'stylistic/semi': ['error', 'never'],
      'stylistic/quotes': ['error', 'single', { avoidEscape: true }],
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': 'off',
      'react-hooks/set-state-in-effect': 'off',
      'react-hooks/refs': 'off',
      'no-empty': ['error', { allowEmptyCatch: true }],
      '@typescript-eslint/no-empty-object-type': [
        'error',
        { allowInterfaces: 'with-single-extends' },
      ],
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
    },
  },
)
