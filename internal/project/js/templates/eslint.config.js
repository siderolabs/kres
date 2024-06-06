// run bun install eslint-plugin-vue typescript-eslint -d for each frontend
// to make the linter work
//@ts-check
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint';

export default [
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/essential'],
  {
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
      "no-console": "off",
      "vue/multi-word-component-names": "off",
      "vue/no-unused-vars": "error"
    },
    plugins: {
      'typescript-eslint': tseslint.plugin,
    },
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        project: './tsconfig.json',
        extraFileExtensions: ['.vue'],
        sourceType: 'module',
      },
    },
  }
]

