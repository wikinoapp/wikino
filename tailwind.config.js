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
          ...require('daisyui/src/theming/themes')['light'],
          primary: 'black',
        },
      },
    ],
  },
};
