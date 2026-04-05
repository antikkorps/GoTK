/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,ts}'],
  theme: {
    extend: {
      colors: {
        base: '#0d1117',
        card: '#161b22',
        accent: '#3fb950',
        'accent-blue': '#58a6ff',
        'text-primary': '#e6edf3',
        'text-muted': '#8b949e',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
    },
  },
  plugins: [],
};
