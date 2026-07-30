import { defineConfig } from 'vite'
import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import { visualizer } from 'rollup-plugin-visualizer'; // Importez le visualizer


// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    babel({ presets: [reactCompilerPreset()] }),
    // Désactivé dans le build du conteneur (CONTAINER_BUILD=1) : pas de navigateur
    // pour `open`, et le report.html ne doit pas finir dans l'image nginx.
    ...(process.env.CONTAINER_BUILD
      ? []
      : [visualizer({
          filename: "./dist/report.html", // Nom du fichier de rapport HTML
          open: true, // Ouvre le rapport automatiquement après le build
          gzipSize: true, // Affiche les tailles gzip
          brotliSize: true, // Affiche les tailles brotli
        })]),
  ],

  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:3333',
        configure: (proxy, _options) => {
          proxy.on('proxyReq', (proxyReq, req, _res) => {
            proxyReq.setHeader('X-Real-IP', req.socket.remoteAddress ?? '127.0.0.1');
          });
        },
      },
    },
  }, build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {

            if (id.includes('@mui/x-date-pickers') || id.includes('@mui/material')) {
              return 'mui-material-libs'; // Un chunk pour les grosses libs
            }
            if (id.includes('@toolpad') || id.includes('@mui')) {
              return 'mui-libs'; // Un chunk pour les grosses libs
            }

            if (id.includes('@tanstack')) {
              return 'tanstack-libs'; // Un chunk pour les grosses libs
            }

            return 'vendor'; // Le reste des dépendances
          }
        },
      },
    },
  },
})