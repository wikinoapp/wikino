const tailwindColors = require("tailwindcss/colors");

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
    "./app/components/**/*.{erb,rb}",
    "./app/javascript/**/*.ts",
    "./app/views/**/*.{erb,rb}",
    "./config/locales/*.yml",
    "./public/*.html",
  ],
  theme: {
    // https://tailwindcss.com/docs/customizing-colors
    colors,
    zIndex: {
      tooltip: 200,
      navbar: 100,
    },
  },
};
