import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import sitemap from '@astrojs/sitemap';

export default defineConfig({
  integrations: [sitemap()],
  site: 'https://antikkorps.github.io',
  base: '/GoTK',
  vite: {
    plugins: [tailwindcss()],
  },
});
