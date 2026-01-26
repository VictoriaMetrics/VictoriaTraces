import eslintReact from "@eslint-react/eslint-plugin";
import typescriptEslint from "@typescript-eslint/eslint-plugin";
import globals from "globals";
import tsParser from "@typescript-eslint/parser";
import path from "node:path";
import { fileURLToPath } from "node:url";
import js from "@eslint/js";
import { FlatCompat } from "@eslint/eslintrc";
import unusedImports from "eslint-plugin-unused-imports";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended,
  allConfig: js.configs.all
});

export default [...compat.extends(
  "eslint:recommended",
  "plugin:@typescript-eslint/recommended",
), eslintReact.configs["recommended-typescript"], {
  plugins: {
    "@typescript-eslint": typescriptEslint,
    "unused-imports": unusedImports,
  },

  languageOptions: {
    globals: {
      ...globals.browser,
    },

    parser: tsParser,
    ecmaVersion: 12,
    sourceType: "module",

    parserOptions: {
      ecmaFeatures: {
        jsx: true,
      },
    },
  },

  rules: {
    "@typescript-eslint/no-unused-expressions": ["error", {
      allowShortCircuit: true,
      allowTernary: true
    }],

    "@typescript-eslint/no-unused-vars": ["error", {
      "argsIgnorePattern": "^_",
      "caughtErrors": "none",
      "caughtErrorsIgnorePattern": "^_",
      "destructuredArrayIgnorePattern": "^_",
      "varsIgnorePattern": "^_",
      "ignoreRestSiblings": true
    }],
    
    "unused-imports/no-unused-imports": "error",

    "object-curly-spacing": [2, "always"],

    indent: ["error", 2, {
      SwitchCase: 1,
    }],

    "linebreak-style": ["error", "unix"],
    quotes: ["error", "double"],
    semi: ["error", "always"],
  },
}];
