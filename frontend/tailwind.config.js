/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        gold: {
          400: '#fbbf24',
          500: '#f5b800',
        },
      },
    },
  },
  plugins: [],
};
