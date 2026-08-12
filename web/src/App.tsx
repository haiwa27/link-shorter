import { useEffect, useState } from "react";

type Link = {
  slug: string;
  ziel: string;
  erstellt: string;
  aufrufe: number;
};

export function App() {
  const [links, setLinks] = useState<Link[]>([]);
  const [ziel, setZiel] = useState("");
  const [wunschSlug, setWunschSlug] = useState("");
  const [fehler, setFehler] = useState<string | null>(null);
  const [laeuft, setLaeuft] = useState(false);

  async function ladeListe() {
    try {
      const antwort = await fetch("/api/links");
      if (!antwort.ok) throw new Error(`Status ${antwort.status}`);
      const daten = await antwort.json();
      setLinks(daten.links ?? []);
    } catch (err) {
      setFehler(err instanceof Error ? err.message : "Unbekannter Fehler");
    }
  }

  useEffect(() => {
    void ladeListe();
  }, []);

  async function anlegen() {
    setLaeuft(true);
    setFehler(null);
    try {
      const antwort = await fetch("/api/links", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ziel, wunschSlug: wunschSlug || undefined }),
      });
      if (!antwort.ok) {
        const koerper = await antwort.json().catch(() => ({}));
        throw new Error(koerper.fehler ?? `Status ${antwort.status}`);
      }
      setZiel("");
      setWunschSlug("");
      await ladeListe();
    } catch (err) {
      setFehler(err instanceof Error ? err.message : "Unbekannter Fehler");
    } finally {
      setLaeuft(false);
    }
  }

  async function loeschen(slug: string) {
    await fetch(`/api/links/${slug}`, { method: "DELETE" });
    await ladeListe();
  }

  return (
    <main>
      <h1>healthgate</h1>
      <p className="hinweis">Kurzlinks anlegen und verwalten.</p>

      {/* Kein form-Element: bewusst ueber Klick-Handler, damit kein
          Seiten-Neuladen die E2E-Tests stoert. */}
      <section className="karte">
        <label htmlFor="ziel">Ziel-URL</label>
        <input
          id="ziel"
          data-testid="eingabe-ziel"
          value={ziel}
          onChange={(e) => setZiel(e.target.value)}
          placeholder="https://www.beispiel.de/eine/lange/adresse"
        />

        <label htmlFor="slug">Wunsch-Slug (optional)</label>
        <input
          id="slug"
          data-testid="eingabe-slug"
          value={wunschSlug}
          onChange={(e) => setWunschSlug(e.target.value)}
          placeholder="mein-link"
        />

        <button data-testid="knopf-anlegen" onClick={anlegen} disabled={laeuft || !ziel}>
          {laeuft ? "Wird angelegt..." : "Kurzlink anlegen"}
        </button>

        {fehler && (
          <p className="fehler" data-testid="fehlermeldung" role="alert">
            {fehler}
          </p>
        )}
      </section>

      <table data-testid="tabelle-links">
        <thead>
          <tr>
            <th>Slug</th>
            <th>Ziel</th>
            <th>Aufrufe</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {links.map((link) => (
            <tr key={link.slug} data-testid={`zeile-${link.slug}`}>
              <td>
                <a href={`/${link.slug}`} data-testid={`link-${link.slug}`}>
                  /{link.slug}
                </a>
              </td>
              <td className="ziel">{link.ziel}</td>
              <td>{link.aufrufe}</td>
              <td>
                <button
                  data-testid={`knopf-loeschen-${link.slug}`}
                  onClick={() => loeschen(link.slug)}
                >
                  Loeschen
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </main>
  );
}
