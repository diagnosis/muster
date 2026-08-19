import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { tanstackRouter} from '@tanstack/router-plugin/vite'
import {resolve} from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
      tanstackRouter({
          target:'react',
          autoCodeSplitting:true,
      }),
      react()],
    resolve:{
      alias:{
          '@': resolve(import.meta.dirname, './src')
      }
    },
    server: {proxy: {'/api':'http://localhost:8088'}},
    preview: { proxy: { '/api': 'http://localhost:8088' } },
})
