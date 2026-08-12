import { expect, test } from "@playwright/test";

test.describe("Betriebsendpunkte", () => {
  test("healthz meldet Bereitschaft mit Version und Slot", async ({ request }) => {
    const antwort = await request.get("/healthz");
    expect(antwort.status()).toBe(200);

    const koerper = await antwort.json();
    expect(koerper.status).toBe("bereit");
    expect(koerper.version).toBeTruthy();
    expect(koerper.slot).toBeTruthy();
  });
});
