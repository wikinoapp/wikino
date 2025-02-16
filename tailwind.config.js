const tailwindColors = require("tailwindcss/colors");
const daisyuiThemes = require("daisyui/src/theming/themes");

const brand = {
  50: "#f9f7f7",
  100: "#f2efee",
  200: "#e6dfdd",
  300: "#d7cdca",
  400: "#bfafaa",
  500: "#a7928c",
  600: "#8f7973",
  700: "#77645e",
  800: "#645450",
  900: "#564a46",
  950: "#2c2523",
};

const colors = {
  inherit: tailwindColors.inherit,
  current: tailwindColors.current,
  transparent: tailwindColors.transparent,
  black: tailwindColors.black,
  white: tailwindColors.white,
  gray: tailwindColors.stone, // 統一感を出すため、灰色はStoneだけを使う
  red: tailwindColors.red,
  orange: tailwindColors.orange,
  amber: tailwindColors.amber,
  yellow: tailwindColors.yellow,
  lime: tailwindColors.lime,
  green: tailwindColors.green,
  emerald: tailwindColors.emerald,
  teal: tailwindColors.teal,
  cyan: tailwindColors.cyan,
  sky: tailwindColors.sky,
  blue: tailwindColors.blue,
  indigo: tailwindColors.indigo,
  violet: tailwindColors.violet,
  purple: tailwindColors.purple,
  fuchsia: tailwindColors.fuchsia,
  pink: tailwindColors.pink,
  rose: tailwindColors.rose,
  brand,
};

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/helpers/**/*.rb",
    "./app/javascript/**/*.ts",
    "./app/views/**/*.erb",
    "./app/views/**/*.rb",
    "./config/locales/*.yml",
    "./lib/wikino/ui/**/*.erb",
    "./lib/wikino/ui/**/*.rb",
    "./public/*.html",
  ],
  theme: {
    // https://tailwindcss.com/docs/customizing-colors
    colors,
    zIndex: {
      navbar: 100,
    },
  },
  plugins: [require("daisyui")],
  daisyui: {
    themes: [
      {
        light: {
          ...daisyuiThemes["light"],
          "base-100": colors.brand[100], // ページ全体の背景色
          "base-200": colors.brand[100], // ナビゲーションバーの背景色
          "base-300": colors.gray[100], // カードの背景色
          "base-content": colors.gray[950], // テキストの色
          "primary-content": colors.gray[100],
          primary: colors.gray[950],
          "secondary-content": colors.gray[950],
          secondary: colors.brand[100],
        },
      },
    ],
  },
};
