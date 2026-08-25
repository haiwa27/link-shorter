import { useCallback, useEffect, useRef, useState } from "react";

type Link = {
  slug: string;
  ziel: string;
  erstellt: string;
  aufrufe: number;
};

type Zustand = {
  status: string;
  version?: string;
  slot?: string;
  grund?: string;
};

const ZUSTAND_INTERVALL = 5000;

export function App() {
  const [links, setLinks] = useState<Link[]>([]);
  const [zustand, setZustand] = useState<Zustand | null>(null);
  const [umgeschaltet, setUmgeschaltet] = useState(false);
  const [ziel, setZiel] = useState("");
  const [wunschSlug, setWunschSlug] = useState("");
  const [fehler, setFehler] = useState<string | null>(null);
  const [laeuft, setLaeuft] = useState(false);
  const [geladen, setGeladen] = useState(false);
  const [zuletztAngelegt, setZuletztAngelegt] = useState<Link | null>(null);
  const [kopiert, setKopiert] = useState(false);

  const vorigerSlot = useRef<string | null>(null);

  const ladeListe = useCallback(async () => {
    try {
      const antwort = await fetch("/api/links");
      if (!antwort.ok) throw new Error(`Die Liste konnte nicht geladen werden (${antwort.status}).`);
      const daten = await antwort.json();
      setLinks(daten.links ?? []);
    } catch (err) {
      setFehler(err instanceof Error ? err.message : "Die Liste konnte nicht geladen werden.");
    } finally {
      setGeladen(true);
    }
  }, []);

  // Zustandsschild: fragt denselben Endpunkt ab, den auch das Health-Gate der
  // Pipeline auswertet. Wechselt der Slot, wird das hier sichtbar -- genau das
  // passiert beim Umschalten und beim Rollback.
  const ladeZustand = useCallback(async () => {
    try {
      const antwort = await fetch("/healthz");
      const daten: Zustand = await antwort.json();
      setZustand(daten);
      if (vorigerSlot.current && daten.slot && daten.slot !== vorigerSlot.current) {
        setUmgeschaltet(true);
        window.setTimeout(() => setUmgeschaltet(false), 6000);
        void ladeListe();
      }
      if (daten.slot) vorigerSlot.current = daten.slot;
    } catch {
      setZustand({ status: "nicht erreichbar" });
    }
  }, [ladeListe]);

  useEffect(() => {
    void ladeListe();
    void ladeZustand();
    const takt = window.setInterval(ladeZustand, ZUSTAND_INTERVALL);
    return () => window.clearInterval(takt);
  }, [ladeListe, ladeZustand]);

  async function anlegen() {
    setLaeuft(true);
    setFehler(null);
    setKopiert(false);
    try {
      const antwort = await fetch("/api/links", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ziel, wunschSlug: wunschSlug || undefined }),
      });
      const daten = await antwort.json().catch(() => null);
      if (!antwort.ok) {
        throw new Error(daten?.fehler ?? `Der Kurzlink konnte nicht angelegt werden (${antwort.status}).`);
      }
      setZuletztAngelegt(daten as Link);
      setZiel("");
      setWunschSlug("");
      await ladeListe();
    } catch (err) {
      setFehler(err instanceof Error ? err.message : "Der Kurzlink konnte nicht angelegt werden.");
    } finally {
      setLaeuft(false);
    }
  }

  async function loeschen(slug: string) {
    setFehler(null);
    try {
      const antwort = await fetch(`/api/links/${slug}`, { method: "DELETE" });
      if (!antwort.ok && antwort.status !== 204) {
        throw new Error(`Der Kurzlink konnte nicht gelöscht werden (${antwort.status}).`);
      }
      if (zuletztAngelegt?.slug === slug) setZuletztAngelegt(null);
      await ladeListe();
    } catch (err) {
      setFehler(err instanceof Error ? err.message : "Der Kurzlink konnte nicht gelöscht werden.");
    }
  }

  async function kopieren(slug: string) {
    try {
      await navigator.clipboard.writeText(`${window.location.origin}/${slug}`);
      setKopiert(true);
      window.setTimeout(() => setKopiert(false), 2000);
    } catch {
      setFehler("Kopieren ist in diesem Browser nicht möglich. Die Adresse steht in der Liste.");
    }
  }

  const slot = zustand?.slot ?? "unbekannt";
  const bereit = zustand?.status === "bereit";
  const slotKlasse = slot === "blue" ? "ist-blau" : slot === "green" ? "ist-gruen" : "ist-unklar";

  return (
    <div className="rahmen">
      {/* Signatur der Oberflaeche: welcher Slot bedient dich gerade. */}
      <div
        className={`schild ${slotKlasse} ${umgeschaltet ? "wechselt" : ""}`}
        role="status"
        aria-live="polite"
      >
        <div className="schild-feld">
          <span className="schild-marke">Slot</span>
          <span className="schild-wert tabular">{slot}</span>
        </div>
        <div className="schild-feld">
          <span className="schild-marke">Version</span>
          <span className="schild-wert tabular">{zustand?.version ?? "—"}</span>
        </div>
        <div className="schild-feld">
          <span className="schild-marke">Zustand</span>
          <span className="schild-wert">
            <i className={`punkt ${bereit ? "punkt-gut" : "punkt-schlecht"}`} aria-hidden="true" />
            {zustand?.status ?? "wird geprüft"}
          </span>
        </div>
        {umgeschaltet && <span className="schild-hinweis">umgeschaltet</span>}
      </div>

      <header className="kopf">
        <h1>healthgate</h1>
        <p>
          Kurzlinks anlegen und verwalten. Das Schild oben zeigt, welcher Slot die Anfrage gerade
          beantwortet — beim Umschalten und beim Rollback wechselt er hier sichtbar.
        </p>
      </header>

      <main className="spalten">
        {/* Kein form-Element: bewusst ueber Klick-Handler, damit kein
            Seiten-Neuladen die E2E-Tests stoert. */}
        <section className="werkbank" aria-labelledby="werkbank-titel">
          <h2 id="werkbank-titel">Neuer Kurzlink</h2>

          <label htmlFor="ziel">Ziel-URL</label>
          <input
            id="ziel"
            data-testid="eingabe-ziel"
            value={ziel}
            onChange={(e) => setZiel(e.target.value)}
            placeholder="https://www.beispiel.de/eine/lange/adresse"
            autoComplete="off"
            spellCheck={false}
          />

          <label htmlFor="slug">
            Wunsch-Slug <span className="beiwerk">optional</span>
          </label>
          <input
            id="slug"
            className="tabular"
            data-testid="eingabe-slug"
            value={wunschSlug}
            onChange={(e) => setWunschSlug(e.target.value)}
            placeholder="mein-link"
            autoComplete="off"
            spellCheck={false}
          />
          <p className="beiwerk">3 bis 32 Zeichen: Buchstaben, Ziffern, Bindestrich, Unterstrich.</p>

          <button data-testid="knopf-anlegen" onClick={anlegen} disabled={laeuft || !ziel}>
            {laeuft ? "Wird angelegt …" : "Kurzlink anlegen"}
          </button>

          {fehler && (
            <p className="fehler" data-testid="fehlermeldung" role="alert">
              {fehler}
            </p>
          )}

          {zuletztAngelegt && !fehler && (
            <div className="quittung">
              <span className="beiwerk">Angelegt</span>
              <code className="tabular">/{zuletztAngelegt.slug}</code>
              <button className="still" onClick={() => kopieren(zuletztAngelegt.slug)}>
                {kopiert ? "Kopiert" : "Kopieren"}
              </button>
            </div>
          )}
        </section>

        <section className="liste" aria-labelledby="liste-titel">
          <div className="liste-kopf">
            <h2 id="liste-titel">Kurzlinks</h2>
            <span className="zahl tabular">{links.length}</span>
          </div>

          {geladen && links.length === 0 ? (
            <p className="leer">Noch keine Kurzlinks. Leg links den ersten an.</p>
          ) : (
            <table data-testid="tabelle-links">
              <thead>
                <tr>
                  <th scope="col">Slug</th>
                  <th scope="col">Ziel</th>
                  <th scope="col" className="rechts">Aufrufe</th>
                  <th scope="col"><span className="versteckt">Aktion</span></th>
                </tr>
              </thead>
              <tbody>
                {links.map((link) => (
                  <tr key={link.slug} data-testid={`zeile-${link.slug}`}>
                    <td>
                      <a
                        className="marke tabular"
                        href={`/${link.slug}`}
                        target="_blank"
                        rel="noreferrer"
                        data-testid={`link-${link.slug}`}
                      >
                        /{link.slug}
                      </a>
                    </td>
                    <td className="ziel" title={link.ziel}>{link.ziel}</td>
                    <td className="rechts tabular">{link.aufrufe}</td>
                    <td className="rechts">
                      <button
                        className="still"
                        data-testid={`knopf-loeschen-${link.slug}`}
                        onClick={() => loeschen(link.slug)}
                      >
                        Löschen
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </main>
    </div>
  );
}
