const tailwindColors = require("tailwindcss/colors");
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

const pageBgColor = "#e6dfdd";

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
    colors,
    zIndex: {
      "global-header": 100,
    },
  },
  plugins: [require("daisyui")],
  daisyui: {
    themes: [
      {
        light: {
          ...daisyuiThemes["light"],
          "base-100": pageBgColor, // ページ全体の背景色
          "base-200": pageBgColor, // ナビゲーションバーの背景色
          "base-300": colors.gray[100], // カードの背景色
          "base-content": colors.gray[950], // テキストの色
          "primary-content": colors.gray[100],
          primary: colors.gray[950],
          "secondary-content": colors.gray[950],
          secondary: pageBgColor,
        },
      },
    ],
  },
};
