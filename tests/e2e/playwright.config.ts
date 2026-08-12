import { defineConfig, devices } from "@playwright/test";

// Basis-URL kommt aus der Umgebung: lokal die Vite-Adresse, in der Pipeline
// die Staging-Adresse. Nie fest verdrahten.
const basisURL = process.env.BASIS_URL ?? "http://localhost:8081";

export default defineConfig({
  testDir: "./specs",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI
    ? [["list"], ["junit", { outputFile: "test-results/junit.xml" }], ["html", { open: "never" }]]
    : [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: basisURL,
    // Spuren und Bilder nur bei Fehlschlag: Story Q-06 braucht sie zur
    // Fehlersuche, ein grüner Lauf soll keine Artefakte anhäufen.
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "off",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
