const colors = require("tailwindcss/colors");
const daisyuiThemes = require("daisyui/src/theming/themes");

const brand = {
  50: "#fae5ff",
  100: "#e9b5fc",
  200: "#d383fa",
  300: "#b753f9",
  400: "#982af8",
  500: "#741ae0",
  600: "#4f13ae",
  700: "#300b7c",
  800: "#17044a",
  900: "#05001a",
};

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/components/**/*.erb",
    "./app/components/**/*.rb",
    "./app/helpers/**/*.rb",
    "./app/javascript/**/*.ts",
    "./app/views/**/*.erb",
    "./config/locales/*.yml",
    "./lib/wikino/ui/**/*.erb",
    "./lib/wikino/ui/**/*.rb",
    "./public/*.html",
  ],
  theme: {
    // https://tailwindcss.com/docs/customizing-colors
    colors: {
      inherit: colors.inherit,
      current: colors.current,
      transparent: colors.transparent,
      black: colors.black,
      white: colors.white,
      gray: colors.slate, // 統一感を出すため、灰色はSlateだけを使う
      red: colors.red,
      orange: colors.orange,
      amber: colors.amber,
      yellow: colors.yellow,
      lime: colors.lime,
      green: colors.green,
      emerald: colors.emerald,
      teal: colors.teal,
      cyan: colors.cyan,
      sky: colors.sky,
      blue: colors.blue,
      indigo: colors.indigo,
      violet: colors.violet,
      purple: colors.purple,
      fuchsia: colors.fuchsia,
      pink: colors.pink,
      rose: colors.rose,
      brand,
    },
  },
  plugins: [require("daisyui")],
  daisyui: {
    themes: [
      {
        light: {
          ...daisyuiThemes["light"],
          "base-100": brand[50], // ページ全体の背景色
          "base-200": colors.slate[800], // サイドバーの背景色
          "base-300": colors.white, // カードの背景色
          "base-content": colors.slate[950], // テキストの色
          "primary-content": colors.slate[100],
          primary: brand[500],
          "secondary-content": colors.slate[100],
          secondary: colors.slate[800],
        },
      },
    ],
  },
};
