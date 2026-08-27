import js from "@eslint/js";
import vitest from "@vitest/eslint-plugin";
import jsxA11y from "eslint-plugin-jsx-a11y";
import playwright from "eslint-plugin-playwright";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import globals from "globals";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "../output", "scripts/lint-colocated-routes.mjs"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommendedTypeChecked],
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      "jsx-a11y": jsxA11y,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.flat["recommended-latest"].rules,
      ...jsxA11y.configs.recommended.rules,
      ...reactRefresh.configs.vite.rules,
      eqeqeq: ["error", "always"],
      "@typescript-eslint/consistent-type-exports": "error",
      "@typescript-eslint/consistent-type-imports": "error",
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/no-misused-promises": "error",
      "@typescript-eslint/switch-exhaustiveness-check": "error",
      "@typescript-eslint/await-thenable": "error",
    },
  },
  {
    files: ["src/**/*.test.{ts,tsx}"],
    plugins: { vitest },
    rules: vitest.configs.recommended.rules,
  },
  {
    ...playwright.configs["flat/recommended"],
    files: ["e2e/**/*.ts"],
    languageOptions: {
      ...playwright.configs["flat/recommended"].languageOptions,
      parser: tseslint.parser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
  },
  {
    extends: [js.configs.recommended, tseslint.configs.disableTypeChecked],
    files: ["scripts/**/*.mjs"],
    languageOptions: {
      ecmaVersion: 2022,
      globals: { ...globals.browser, ...globals.node },
    },
  },
  {
    files: ["src/hooks/use-data-table.ts"],
    rules: {
      // TanStack Table intentionally exposes non-memoizable functions.
      "react-hooks/incompatible-library": "off",
    },
  },
  {
    files: ["src/types/tanstack-table.d.ts"],
    rules: {
      // Declaration merging must repeat TanStack Table's generic parameters.
      "@typescript-eslint/no-unused-vars": "off",
    },
  },
  {
    files: ["src/routes/__root.tsx", "src/routes/**/index.tsx"],
    rules: {
      "react-refresh/only-export-components": [
        "error",
        { allowConstantExport: true, allowExportNames: ["Route"] },
      ],
    },
  },
  {
    files: ["src/components/data-table/data-table-pagination.tsx"],
    rules: {
      "react-refresh/only-export-components": ["error", { allowExportNames: ["PAGE_SIZE_OPTIONS"] }],
    },
  },
  {
    files: ["src/components/ui/button.tsx"],
    rules: {
      "react-refresh/only-export-components": ["error", { allowExportNames: ["buttonVariants"] }],
    },
  },
  {
    files: ["src/components/ui/dropdown-menu.tsx"],
    rules: {
      "react-refresh/only-export-components": [
        "error",
        {
          allowExportNames: [
            "DropdownMenu",
            "DropdownMenuRadioGroup",
            "DropdownMenuTrigger",
          ],
        },
      ],
    },
  },
  {
    files: ["src/components/ui/sidebar.tsx"],
    rules: {
      "react-refresh/only-export-components": ["error", { allowExportNames: ["useSidebar"] }],
    },
  },
  {
    files: ["src/lib/i18n.tsx"],
    rules: {
      "react-refresh/only-export-components": [
        "error",
        {
          allowConstantExport: true,
          allowExportNames: [
            "LOCALE_OPTIONS",
            "LOCALE_STORAGE_KEY",
            "apiErrorMessage",
            "formatStatus",
            "hasTranslation",
            "normalizeLocale",
            "translate",
            "useI18n",
          ],
        },
      ],
    },
  },
  {
    files: ["src/lib/theme.tsx"],
    rules: {
      "react-refresh/only-export-components": [
        "error",
        {
          allowConstantExport: true,
          allowExportNames: [
            "THEME_STORAGE_KEY",
            "applyTheme",
            "normalizeTheme",
            "resolveTheme",
            "useTheme",
          ],
        },
      ],
    },
  },
);
