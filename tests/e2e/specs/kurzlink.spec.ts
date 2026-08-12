import { expect, test } from "@playwright/test";

// Deckt Stories P-01 bis P-04 ab. Bewusst ein durchgehender Ablauf statt
// isolierter Klicks: geprüft wird, was der Nutzer tatsächlich tut.

test("Kurzlink anlegen, in der Liste sehen, folgen und loeschen", async ({ page }) => {
  const slug = `test${Date.now()}`;
  const ziel = "https://www.beispiel.de/ein/langer/pfad";

  await page.goto("/");

  // Anlegen (P-01, P-05)
  await page.getByTestId("eingabe-ziel").fill(ziel);
  await page.getByTestId("eingabe-slug").fill(slug);
  await page.getByTestId("knopf-anlegen").click();

  // In der Liste sichtbar (P-03)
  const zeile = page.getByTestId(`zeile-${slug}`);
  await expect(zeile).toBeVisible();
  await expect(zeile).toContainText(ziel);

  // Weiterleitung folgen (P-02). Die Umleitung führt nach aussen, daher wird
  // nur die Antwort geprüft und nicht navigiert.
  const antwort = await page.request.get(`/${slug}`, { maxRedirects: 0 });
  expect(antwort.status()).toBe(302);
  expect(antwort.headers()["location"]).toBe(ziel);

  // Loeschen (P-04)
  await page.getByTestId(`knopf-loeschen-${slug}`).click();
  await expect(page.getByTestId(`zeile-${slug}`)).toHaveCount(0);

  const danach = await page.request.get(`/${slug}`, { maxRedirects: 0 });
  expect(danach.status()).toBe(404);
});

test("ungueltige Ziel-URL wird abgelehnt", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("eingabe-ziel").fill("nur-irgendein-text");
  await page.getByTestId("knopf-anlegen").click();

  await expect(page.getByTestId("fehlermeldung")).toBeVisible();
});

test("belegter Wunsch-Slug ergibt eine Fehlermeldung", async ({ page }) => {
  const slug = `doppelt${Date.now()}`;

  await page.goto("/");
  for (let i = 0; i < 2; i++) {
    await page.getByTestId("eingabe-ziel").fill("https://beispiel.de");
    await page.getByTestId("eingabe-slug").fill(slug);
    await page.getByTestId("knopf-anlegen").click();
  }

  await expect(page.getByTestId("fehlermeldung")).toBeVisible();
});
