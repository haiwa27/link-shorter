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

const TAKT = 5000;
const PROBEN_MAX = 32;

function dauerText(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = String(s % 60).padStart(2, "0");
  const mm = String(m).padStart(2, "0");
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
}

export function App() {
  const [links, setLinks] = useState<Link[]>([]);
  const [zustand, setZustand] = useState<Zustand | null>(null);
  const [proben, setProben] = useState<number[]>([]);
  const [umgeschaltet, setUmgeschaltet] = useState(false);
  const [seitWechsel, setSeitWechsel] = useState<number>(() => Date.now());
  const [jetzt, setJetzt] = useState<number>(() => Date.now());
  const [ziel, setZiel] = useState("");
  const [wunschSlug, setWunschSlug] = useState("");
  const [fehler, setFehler] = useState<string | null>(null);
  const [laeuft, setLaeuft] = useState(false);
  const [geladen, setGeladen] = useState(false);
  const [neuerSlug, setNeuerSlug] = useState<string | null>(null);
  const [kopiertSlug, setKopiertSlug] = useState<string | null>(null);

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

  // Fragt denselben Endpunkt ab, den auch das Health-Gate der Pipeline
  // auswertet. Die Antwortzeit jeder Probe wird gemessen und als Verlauf
  // gezeichnet: echte Messwerte, keine Dekoration.
  const ladeZustand = useCallback(async () => {
    const beginn = performance.now();
    try {
      const antwort = await fetch("/healthz");
      const dauer = performance.now() - beginn;
      const daten: Zustand = await antwort.json();
      setZustand(daten);
      setProben((alt) => [...alt.slice(-(PROBEN_MAX - 1)), dauer]);
      if (vorigerSlot.current && daten.slot && daten.slot !== vorigerSlot.current) {
        setUmgeschaltet(true);
        setSeitWechsel(Date.now());
        window.setTimeout(() => setUmgeschaltet(false), 7000);
        void ladeListe();
      }
      if (daten.slot) vorigerSlot.current = daten.slot;
    } catch {
      setZustand({ status: "nicht erreichbar" });
      setProben((alt) => [...alt.slice(-(PROBEN_MAX - 1)), 0]);
    }
  }, [ladeListe]);

  useEffect(() => {
    void ladeListe();
    void ladeZustand();
    const abfrage = window.setInterval(ladeZustand, TAKT);
    const uhr = window.setInterval(() => setJetzt(Date.now()), 1000);
    return () => {
      window.clearInterval(abfrage);
      window.clearInterval(uhr);
    };
  }, [ladeListe, ladeZustand]);

  const slot = zustand?.slot ?? "";
  const bereit = zustand?.status === "bereit";

  // Der Farbblock der Seite ist der bedienende Slot. Beim Umschalten und beim
  // Rollback wechselt er sichtbar, ohne Neuladen.
  useEffect(() => {
    document.documentElement.dataset.slot = slot || "unbekannt";
  }, [slot]);

  async function anlegen() {
    setLaeuft(true);
    setFehler(null);
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
      setNeuerSlug((daten as Link).slug);
      window.setTimeout(() => setNeuerSlug(null), 2500);
      setZiel("");
      setWunschSlug("");
      await ladeListe();
    } catch (err) {
      setFehler(err instanceof Error ? err.message : "Der Kurzlink konnte nicht angelegt werden.");
    } finally {
      setLaeuft(false);
    }
  }

  async function loeschen(slug_: string) {
    setFehler(null);
    try {
      const antwort = await fetch(`/api/links/${slug_}`, { method: "DELETE" });
      if (!antwort.ok && antwort.status !== 204) {
        throw new Error(`Der Kurzlink konnte nicht gelöscht werden (${antwort.status}).`);
      }
      await ladeListe();
    } catch (err) {
      setFehler(err instanceof Error ? err.message : "Der Kurzlink konnte nicht gelöscht werden.");
    }
  }

  async function kopieren(slug_: string) {
    try {
      await navigator.clipboard.writeText(`${window.location.origin}/${slug_}`);
      setKopiertSlug(slug_);
      window.setTimeout(() => setKopiertSlug(null), 1800);
    } catch {
      setFehler("Kopieren geht nur über HTTPS oder localhost.");
    }
  }

  function beiEnter(e: React.KeyboardEvent) {
    if (e.key === "Enter" && ziel && !laeuft) void anlegen();
  }

  const letzteProbe = proben.length > 0 ? proben[proben.length - 1] : 0;
  const probenMax = Math.max(60, ...proben);

  return (
    <div className="blatt">
      <header className="kopf">
        <span className="wortmarke">healthgate</span>
        <span className="kopf-fakt tabular">{zustand?.version ?? "?"}</span>
      </header>

      <section className="band" aria-live="polite">
        <div className="band-wort">
          <span className="band-marke">Bedienender Slot</span>
          <h1 className="slotwort">{slot || "?"}</h1>
        </div>
        <div className="band-fakten">
          <div className="fakt">
            <span className="band-marke">Zustand</span>
            <span className="fakt-wert">{zustand?.status ?? "wird geprüft"}</span>
          </div>
          <div className="fakt">
            <span className="band-marke">Seit Wechsel</span>
            <span className="fakt-wert tabular">{dauerText(jetzt - seitWechsel)}</span>
          </div>
          <div className="fakt">
            <span className="band-marke">Antwortzeit</span>
            <span className="fakt-wert tabular">{Math.round(letzteProbe)} ms</span>
          </div>
          <div className="fakt funken-fakt" aria-hidden="true">
            <svg className="funken" viewBox={`0 0 ${PROBEN_MAX * 5} 30`} preserveAspectRatio="none">
              {proben.map((p, i) => {
                const hoehe = Math.max(2, (p / probenMax) * 28);
                return (
                  <rect
                    key={i}
                    x={i * 5}
                    y={30 - hoehe}
                    width={3.2}
                    height={hoehe}
                    className={i === proben.length - 1 ? "funke aktuell" : "funke"}
                  />
                );
              })}
            </svg>
          </div>
        </div>
        <div className="band-status">
          {!bereit && zustand && <span className="band-warnhinweis">{zustand.status}</span>}
          {umgeschaltet && <span className="band-wechsel">Umgeschaltet</span>}
        </div>
      </section>

      {/* Kein form-Element: bewusst über Klick-Handler, damit kein
          Seiten-Neuladen die E2E-Tests stört. */}
      <section className="abschnitt" aria-label="Kurzlink anlegen">
        <div className="abschnitt-kopf">
          <h2>Neuer Kurzlink</h2>
        </div>
        <div className="zeile-anlegen">
          <div className="feld waechst">
            <label htmlFor="ziel">Ziel-URL</label>
            <input
              id="ziel"
              data-testid="eingabe-ziel"
              value={ziel}
              onChange={(e) => setZiel(e.target.value)}
              onKeyDown={beiEnter}
              placeholder="https://www.beispiel.de/eine/lange/adresse"
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <div className="feld schmal">
            <label htmlFor="slug">Wunsch-Slug</label>
            <input
              id="slug"
              className="tabular"
              data-testid="eingabe-slug"
              value={wunschSlug}
              onChange={(e) => setWunschSlug(e.target.value)}
              onKeyDown={beiEnter}
              placeholder="optional"
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <button data-testid="knopf-anlegen" onClick={anlegen} disabled={laeuft || !ziel}>
            {laeuft ? "Wird angelegt" : "Anlegen"}
          </button>
        </div>
        {fehler && (
          <p className="fehler" data-testid="fehlermeldung" role="alert">
            {fehler}
          </p>
        )}
      </section>

      <section className="abschnitt" aria-labelledby="bestand-titel">
        <div className="abschnitt-kopf">
          <h2 id="bestand-titel">Kurzlinks</h2>
          <span className="anzahl tabular">{links.length}</span>
        </div>

        {geladen && links.length === 0 ? (
          <p className="leer">Noch keine Kurzlinks.</p>
        ) : (
          <table data-testid="tabelle-links">
            <thead>
              <tr>
                <th scope="col">Slug</th>
                <th scope="col">Ziel</th>
                <th scope="col" className="rechts">Aufrufe</th>
                <th scope="col"><span className="versteckt">Aktionen</span></th>
              </tr>
            </thead>
            <tbody>
              {links.map((link) => (
                <tr
                  key={link.slug}
                  data-testid={`zeile-${link.slug}`}
                  className={link.slug === neuerSlug ? "frisch" : ""}
                >
                  <td>
                    <a
                      className="slug tabular"
                      href={`/${link.slug}`}
                      target="_blank"
                      rel="noreferrer"
                      data-testid={`link-${link.slug}`}
                    >
                      /{link.slug}
                    </a>
                  </td>
                  <td className="ziel" title={link.ziel}>{link.ziel}</td>
                  <td className="rechts tabular aufrufe">{link.aufrufe}</td>
                  <td className="rechts aktionen">
                    <button className="still" onClick={() => kopieren(link.slug)}>
                      {kopiertSlug === link.slug ? "Kopiert" : "Kopieren"}
                    </button>
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
    </div>
  );
}
