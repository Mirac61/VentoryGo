import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath, URL } from 'node:url'

// https://vite.dev/config/
export default defineConfig({
    plugins: [react()],
    server: {
        // dev-proxy/proxy.js leitet fest auf 5173. Ohne strictPort würde Vite bei
        // belegtem Port stillschweigend auf 5174 ausweichen und der Proxy liefe
        // ins Leere — lieber laut scheitern als ein halb funktionierender Stack.
        port: 5173,
        strictPort: true,
        proxy: {
            '/api': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
        },
        fs: {
            // shared/ liegt als Geschwisterordner außerhalb von frontend/, Vite
            // blockiert das per Default (@fs/-Zugriffe außerhalb der Projektwurzel).
            allow: [fileURLToPath(new URL('..', import.meta.url))],
        },
    },
})