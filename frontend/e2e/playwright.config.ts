import { defineConfig, devices } from '@playwright/test'
import { fileURLToPath } from 'node:url'
import * as dotenv from 'dotenv';

dotenv.config({ path: fileURLToPath(new URL('.env', import.meta.url)) })

const frontendRoot = fileURLToPath(new URL('..', import.meta.url))
const CI = !!process.env.CI

export default defineConfig({
    testDir: './specs',
    fullyParallel: true,
    forbidOnly: CI,               // stray .only fails the CI job
    retries: CI ? 2 : 0,
    reporter: CI ? 'github' : 'list',

    use: {
        baseURL: 'http://localhost:5173',
        trace: 'on-first-retry',
    },

    projects: [
        { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
        { name: 'mobile', use: {...devices['iPhone 17 Pro'], browserName: 'chromium'}}
    ],

    webServer: {
        command: CI ? 'npm run build && npm run preview -- --port 5173' : 'npm run dev',
        cwd:frontendRoot,
        url: 'http://localhost:5173',
        reuseExistingServer: !CI,   // local: reuse your running dev server; CI: always fresh
        timeout: 120_000,
    },
})