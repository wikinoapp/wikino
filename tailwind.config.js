const colors = require('tailwindcss/colors');
const daisyuiThemes = require('daisyui/src/theming/themes')

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './app/components/**/*.erb',
    './app/components/**/*.rb',
    './app/helpers/**/*.rb',
    './app/javascript/**/*.ts',
    './app/views/**/*.erb',
    './config/locales/*.yml',
    './lib/wikino/ui/**/*.erb',
    './lib/wikino/ui/**/*.rb',
    './public/*.html',
  ],
  theme: {
    extend: {},
  },
  plugins: [
    require('daisyui'),
  ],
  daisyui: {
    themes: [
      {
        light: {
          ...daisyuiThemes['light'],
          'base-100': colors.fuchsia[50],       // ページ全体の背景色
          'base-200': '#0e0d25',                // サイドバーの背景色
          'base-300': colors.white,             // カードの背景色
          'base-content': colors.slate[950],    // テキストの色
          'primary-content': colors.violet[50],
          'primary': colors.violet[600],
        },
      },
    ],
  },
};
