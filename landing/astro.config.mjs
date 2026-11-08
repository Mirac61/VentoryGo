// @ts-check
import { defineConfig } from 'astro/config';

// https://astro.build/config
import { fileURLToPath } from 'node:url'

import react from '@astrojs/react';

export default defineConfig({
  server: { port: 4173 },

  vite: {
    server: {
      strictPort: true,
      fs: { allow: [fileURLToPath(new URL('..', import.meta.url))] },
    },
  },

  integrations: [react()],
})