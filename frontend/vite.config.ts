import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath, URL } from 'node:url'

// https://vite.dev/config/
export default defineConfig({
    plugins: [react()],
    server: {
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