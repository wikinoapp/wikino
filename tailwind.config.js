/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './app/components/**/*.erb',
    './app/components/**/*.rb',
    './app/helpers/**/*.rb',
    './app/javascript/**/*.ts',
    './app/views/**/*.erb',
    './config/locales/*.yml',
    './lib/nonoto/ui/**/*.erb',
    './lib/nonoto/ui/**/*.rb',
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
        dark: {
          ...require('daisyui/src/theming/themes')['dark'],
          'base-100': '#0c0a09',
          'base-200': '#1c1917',
          'base-300': '#292524',
          primary: 'white',
          secondary: '#ec4899',
        }
      },
    ],
  },
};
