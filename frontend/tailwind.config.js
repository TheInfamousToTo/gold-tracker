/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        // Warm, slightly brown darks — patinated metal rather than
        // neutral black, so gold sits in the same world as its ground.
        ink: {
          DEFAULT: '#12100C',
          raised: '#1A1712',
          sunken: '#0C0A07',
        },
        line: {
          DEFAULT: '#2A251C',
          bright: '#3A3326',
        },
        gold: {
          100: '#F0DFAE', // champagne, for figures that carry weight
          400: '#E8B33A',
          500: '#D19A22',
          600: '#A87715',
        },
        // Verdigris and oxide: what metal actually does over time, and
        // quieter than stock emerald/red against a warm ground.
        patina: '#6FA88A',
        oxide: '#C4635A',
        chalk: '#EDE7DA',
        muted: '#8A8171',
      },
      fontFamily: {
        display: ['Archivo', 'Arial Narrow', 'system-ui', 'sans-serif'],
        sans: ['"Inter Tight"', 'Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'Consolas', 'monospace'],
      },
      letterSpacing: {
        stamp: '0.18em',
      },
      borderRadius: {
        chip: '4px',
      },
    },
  },
  plugins: [],
};
