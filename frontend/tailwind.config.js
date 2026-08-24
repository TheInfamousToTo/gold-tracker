/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        // Neutral, low-chroma ground. Visual management only works if the
        // background is silent: any colour on the page has to read as a
        // signal, and it cannot do that against a coloured surface.
        ink: {
          DEFAULT: '#0F1215',
          raised: '#161A1F',
          sunken: '#0A0C0F',
        },
        line: {
          DEFAULT: '#232A31',
          bright: '#333C45',
        },
        chalk: '#E8ECEF',
        muted: '#8B949E',

        // Andon colours. These three are reserved: they mean normal,
        // watch, and abnormal, and they are never used for decoration.
        // `DEFAULT` fills and borders, `bright` carries text on dark.
        ok: {
          DEFAULT: '#16A34A',
          bright: '#4ADE80',
        },
        warn: {
          DEFAULT: '#D97706',
          bright: '#FBBF24',
        },
        bad: {
          DEFAULT: '#DC2626',
          bright: '#F87171',
        },

        // Identity only — the masthead mark. Never a figure, never a
        // status, never a control, or it starts competing with the andon.
        brand: '#D9A93B',
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
