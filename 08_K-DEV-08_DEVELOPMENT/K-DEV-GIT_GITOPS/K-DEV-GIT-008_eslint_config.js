// ─────────────────────────────────────────────────────────────────────────────
// Kubric-UiDR — ESLint Configuration (Next.js / TypeScript)
// ─────────────────────────────────────────────────────────────────────────────

/** @type {import('eslint').Linter.Config} */
module.exports = {
  root: true,

  // ── Parser ───────────────────────────────────────────────────────────────
  parser: "@typescript-eslint/parser",
  parserOptions: {
    ecmaVersion: "latest",
    sourceType: "module",
    ecmaFeatures: { jsx: true },
    project: "./tsconfig.json",
  },

  // ── Extends ──────────────────────────────────────────────────────────────
  extends: [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:@typescript-eslint/recommended-type-checked",
    "plugin:react/recommended",
    "plugin:react-hooks/recommended",
    "plugin:jsx-a11y/recommended",
    "next/core-web-vitals",
    "prettier",
  ],

  // ── Plugins ──────────────────────────────────────────────────────────────
  plugins: ["@typescript-eslint", "react", "jsx-a11y", "import"],

  // ── Settings ─────────────────────────────────────────────────────────────
  settings: {
    react: { version: "detect" },
    "import/resolver": {
      typescript: { alwaysTryTypes: true },
    },
  },

  // ── Environment ──────────────────────────────────────────────────────────
  env: {
    browser: true,
    es2024: true,
    node: true,
    jest: true,
  },

  // ── Rules ────────────────────────────────────────────────────────────────
  rules: {
    // ── TypeScript strict ────────────────────────────────────────────────
    "@typescript-eslint/no-explicit-any": "error",
    "@typescript-eslint/no-unused-vars": [
      "error",
      { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
    ],
    "@typescript-eslint/consistent-type-imports": [
      "error",
      { prefer: "type-imports", fixStyle: "inline-type-imports" },
    ],
    "@typescript-eslint/no-floating-promises": "error",
    "@typescript-eslint/no-misused-promises": [
      "error",
      { checksVoidReturn: { attributes: false } },
    ],
    "@typescript-eslint/strict-boolean-expressions": "warn",
    "@typescript-eslint/switch-exhaustiveness-check": "error",
    "@typescript-eslint/prefer-nullish-coalescing": "warn",
    "@typescript-eslint/no-unnecessary-condition": "warn",

    // ── React ────────────────────────────────────────────────────────────
    "react/react-in-jsx-scope": "off",
    "react/prop-types": "off",
    "react/self-closing-comp": "error",
    "react/jsx-sort-props": [
      "warn",
      { callbacksLast: true, shorthandFirst: true, reservedFirst: true },
    ],
    "react-hooks/rules-of-hooks": "error",
    "react-hooks/exhaustive-deps": "warn",

    // ── Accessibility ────────────────────────────────────────────────────
    "jsx-a11y/anchor-is-valid": "error",
    "jsx-a11y/click-events-have-key-events": "error",
    "jsx-a11y/no-noninteractive-element-interactions": "warn",

    // ── Import ordering ──────────────────────────────────────────────────
    "import/order": [
      "error",
      {
        groups: [
          "builtin",
          "external",
          "internal",
          "parent",
          "sibling",
          "index",
          "type",
        ],
        "newlines-between": "always",
        alphabetize: { order: "asc", caseInsensitive: true },
      },
    ],
    "import/no-duplicates": "error",

    // ── General ──────────────────────────────────────────────────────────
    "no-console": ["warn", { allow: ["warn", "error"] }],
    "no-debugger": "error",
    "prefer-const": "error",
    eqeqeq: ["error", "always"],
    curly: ["error", "all"],
  },

  // ── Overrides ────────────────────────────────────────────────────────────
  overrides: [
    {
      files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}"],
      env: { jest: true },
      rules: {
        "@typescript-eslint/no-explicit-any": "off",
        "@typescript-eslint/no-floating-promises": "off",
      },
    },
    {
      files: ["*.config.{js,ts,mjs}", "next.config.mjs"],
      rules: {
        "@typescript-eslint/no-var-requires": "off",
      },
    },
  ],

  // ── Ignore patterns ──────────────────────────────────────────────────────
  ignorePatterns: [
    "node_modules/",
    ".next/",
    "out/",
    "coverage/",
    "*.min.js",
    "public/",
  ],
};
