# healthgate

Continuous-Delivery-Strecke, die ein fehlerhaftes Release selbst erkennt und ohne
menschliches Eingreifen zurückrollt. Nach dem Umschalten auf eine neue Version
beobachtet die Pipeline die Anwendung für ein definiertes Zeitfenster anhand
echter Metriken; überschreitet die Fehlerrate den Grenzwert, schwenkt sie auf die
vorherige Version zurück. Die mitgelieferte Anwendung — ein Link-Shortener — ist
absichtlich klein und dient als Objekt, das ausgeliefert, überwacht und im
Fehlerfall zurückgeholt wird.

**Ein Health-Check sagt, ob der Prozess lebt — nicht, ob das Release taugt.**

**Stack:** Go 1.22 · net/http (Standardbibliothek) · PostgreSQL 16 · React 18 +
Vite + TypeScript · Caddy 2 · Docker Compose (Blue/Green) · Prometheus + Grafana ·
Jenkins · Playwright · ohne Authentifizierung (Scope-Entscheidung)

> Keine CI-Badges: die Pipeline läuft nach Projektvorgabe auf Jenkins, nicht auf
> GitHub Actions. Der Build-Status ist in der Jenkins-Oberfläche einsehbar; ein
> Badge würde einen Workflow anzeigen, den es hier nicht gibt.

---

## Inhalt

1. [Bausteine](#bausteine)
2. [Architektur](#architektur)
3. [Domänenmodell](#domänenmodell)
4. [Projektstruktur](#projektstruktur)
5. [Schnellstart (Entwicklung)](#schnellstart-entwicklung)
6. [Konfiguration](#konfiguration)
7. [Betriebsskripte](#betriebsskripte)
8. [API](#api)
9. [Tests und Qualitätssicherung](#tests-und-qualitätssicherung)
10. [CI/CD und Deployment](#cicd-und-deployment)
11. [Sicherheit](#sicherheit)
12. [Umsetzungsstand](#umsetzungsstand)

---

## Bausteine

| Baustein | Ort | Kurzbeschreibung |
|---|---|---|
| Anwendung | `app/` | Link-Shortener: anlegen, auflösen, auflisten, löschen |
| Betriebsendpunkte | `app/internal/handler`, `app/internal/metrics` | `/healthz` und `/metrics` als Grundlage für Health-Gate und Dashboards |
| Chaos-Schalter | `app/internal/chaos` | erzeugt reproduzierbar Fehlerantworten für die Vorführung |
| Blue/Green | `deploy/` | zwei Produktionsslots, atomares Umschalten am Reverse Proxy |
| Health-Gate | `deploy/scripts/observe.sh` | Beobachtungsfenster mit Prometheus-Abfrage, Exitcode steuert den Rollback |
| Pipeline | `Jenkinsfile` | Build, Tests, Staging, E2E, Freigabe, Umschalten, Beobachtung, Rollback |
| Observability | `monitoring/` | Prometheus mit Alarmregeln, Grafana mit Datenquelle und Dashboard als Datei |
| E2E-Tests | `tests/e2e/` | Playwright gegen Staging, läuft als Gate vor Produktion |

**Health-Gate.** Der Kern des Projekts. `observe.sh` fragt in festen Abständen
den Anteil der 5xx-Antworten des neu aktivierten Slots bei Prometheus ab. Bleibt
er unter dem Grenzwert, gilt das Deployment als bestätigt; überschreitet er ihn,
endet das Skript mit Exitcode 1 und die Pipeline löst im `post`-Block den
Rückschwenk aus.

**Blue/Green.** Beide Slots laufen dauerhaft, nur einer bekommt Verkehr. Der
Rollback ist damit eine Umlenkung und kein Neubau — Sekunden statt Minuten.

**Chaos-Schalter.** Die Fehlerrate kommt ausschließlich aus einer
Umgebungsvariable und ist zur Laufzeit nicht umschaltbar; es gibt keinen
Endpunkt, über den sich von außen Fehler auslösen ließen. Für die Vorführung
wird eine Version mit gesetzter Rate ausgeliefert.

**Merksatz:** `/healthz` und `/metrics` sind vom Chaos-Schalter ausgenommen.
Bliebe der Health-Check ebenfalls rot, bräche die Pipeline schon vor dem
Umschalten ab und der metrikbasierte Rollback käme nie zum Einsatz.

---

## Architektur

    Browser (React SPA, Vite-Build)
        │
    Caddy ── Reverse Proxy, Umschalten per Admin-API (:2019), /metrics gesperrt
        │
        ├── app-blue  (Go, :8080) ──┐
        └── app-green (Go, :8080) ──┴── PostgreSQL 16
                │
                └── /metrics ──▶ Prometheus (:9090) ──▶ Grafana (:3000)
                                      │
                                      ▼
                              Jenkins (Health-Gate)
                                      │
                                      └── switch-slot.sh / rollback.sh ──▶ Caddy

### Tragende Grundsätze

1. **Die Anwendung ist zustandslos.** Beide Slots sprechen dieselbe Datenbank.
   Ohne das wäre ein Umschalten mit Datenverlust verbunden.
2. **Der aktive Slot steht in genau einer Datei.** `deploy/caddy/active-slot.conf`
   im Deployment-Verzeichnis ist die einzige Quelle der Wahrheit;
   `active-slot.sh` liest sie, `switch-slot.sh` schreibt sie. Sie ist
   Laufzeitzustand und deshalb nicht versioniert — im Repository liegt nur
   `active-slot.conf.vorlage` (Entscheidung E-015).
3. **Das Umschalten unterbricht keine Verbindung.** Caddy übernimmt die neue
   Konfiguration über die Admin-Schnittstelle; die Anwendung räumt beim Beenden
   mit zehn Sekunden Auslaufzeit ab.
4. **Der Rückweg ist kürzer als der Hinweg.** Der alte Slot läuft weiter, ein
   Rollback baut nichts neu.
5. **Metriken entscheiden, nicht der Health-Check.** Der häufigste Fall eines
   schlechten Releases liegt dazwischen: der Prozess lebt, meldet sich gesund und
   liefert trotzdem Fehler aus.
6. **Kein Label enthält Nutzereingaben.** Pfade werden auf eine feste Menge von
   Bezeichnern abgebildet, bevor sie als Prometheus-Label auftauchen.

---

## Domänenmodell

### Link

| Feld | Art | Bedeutung |
|---|---|---|
| `slug` | Text, Primärschlüssel | Kurzbezeichner im Pfad |
| `ziel` | Text | vollständige Ziel-URL |
| `erstellt` | Zeitstempel UTC | Anlagezeitpunkt |
| `aufrufe` | Ganzzahl | Zähler der Weiterleitungen |

Slug-Regeln (`app/internal/shortener`):

| Regel | Wert |
|---|---|
| Länge | 3 bis 32 Zeichen |
| Erlaubte Zeichen | `A–Z`, `a–z`, `0–9`, `-`, `_` |
| Erzeugte Länge | 7 Zeichen, `crypto/rand` |
| Alphabet erzeugter Slugs | ohne `0`, `O`, `I`, `l` |
| Reserviert | `api`, `healthz`, `metrics`, `assets`, `admin`, `static`, `favicon.ico` |
| Erlaubte Ziel-Schemata | `http`, `https`, Host erforderlich |

**Merksatz:** Erzeugte Slugs meiden verwechselbare Zeichen, damit ein
vorgelesener oder abgetippter Kurzlink funktioniert. Reservierte Slugs
verhindern, dass ein Wunsch-Slug eigene Pfade verdeckt.

### Deployment-Statusmaschine

| Schritt | Prüfung | Bei Erfolg | Bei Fehlschlag |
|---|---|---|---|
| Zielslot bespielen | Container startet | weiter | Pipeline bricht ab, Verkehr unverändert |
| Vor dem Umschalten | dreimal `/healthz` in Folge grün | weiter | Pipeline bricht ab, Verkehr unverändert |
| Umschalten | Konfiguration wirksam, Slot bestätigt | Verkehr auf neuem Slot | Abbruch, kein Rollback nötig |
| Beobachtungsfenster | 5xx-Anteil unter Grenzwert | Deployment bestätigt | `rollback.sh`, Build rot |
| Rollback | vorheriger Slot dreimal grün | Verkehr zurück | manueller Eingriff nötig |

Die Historie liegt als `deploy/state/historie.tsv` mit Zeitpunkt, Slot, Ergebnis
und Grund. `deploy/state/vorheriger-slot` hält das Rollback-Ziel.

**Merksatz:** Ein zurückgerolltes Deployment ist kein Erfolg, sondern ein
verhinderter Schaden. Der Build bleibt deshalb rot.

### Metriken

| Metrik | Labels |
|---|---|
| `healthgate_http_requests_total` | `method`, `route`, `status`, `slot` |
| `healthgate_http_request_duration_seconds_sum` | `route`, `slot` |
| `healthgate_http_request_duration_seconds_count` | `route`, `slot` |
| `healthgate_build_info` | `version`, `slot` |
| `healthgate_uptime_seconds` | `slot` |

Werte von `route`: `/`, `/healthz`, `/metrics`, `/api/links`, `/assets`, `/:slug`.

---

## Projektstruktur

    .
    ├── app/
    │   ├── cmd/server/            Einstiegspunkt: Konfiguration, Middleware-Kette, Shutdown
    │   ├── internal/
    │   │   ├── chaos/             erzeugt Fehlerantworten; /healthz bleibt ausgenommen
    │   │   ├── config/            Umgebungsvariablen, Fail-fast bei Pflichtwerten in prod
    │   │   ├── handler/           HTTP-Endpunkte; Weiterleitung fällt bei Nichttreffer auf Frontend zurück
    │   │   ├── metrics/           Prometheus-Textformat ohne externe Abhängigkeit
    │   │   ├── shortener/         Fachlogik ohne Abhängigkeiten — unit-getestet
    │   │   └── store/             Datenzugriff hinter einer Schnittstelle; PostgreSQL oder In-Memory
    │   ├── migrations/            SQL, wird beim Start von PostgreSQL eingelesen
    │   └── Dockerfile             mehrstufig: Frontend, Backend, Alpine-Laufzeitbild
    ├── web/                       React SPA; Vite proxyt /api im Entwicklungsbetrieb
    ├── deploy/
    │   ├── caddy/                 Caddyfile und Vorlage für active-slot.conf
    │   ├── scripts/               Umschalten, Warten, Beobachten, Rollback, Lastgenerierung
    │   ├── state/                 Laufzeitzustand: vorheriger Slot, Historie (nicht versioniert)
    │   ├── docker-compose.staging.yml
    │   └── docker-compose.prod.yml   beide Slots plus Caddy und PostgreSQL
    ├── monitoring/                Prometheus, Alarmregeln, Grafana-Provisionierung
    ├── tests/e2e/                 Playwright; Basis-URL kommt aus der Umgebung
    ├── tools/                     setup.sh, story-start.sh, story-pr.sh, release.sh
    ├── .githooks/                 aktiv über core.hooksPath, gelten für alle im Team
    ├── docs/                      Architektur, Entscheidungen, Git-Konventionen, Übergabe
    └── Jenkinsfile                Pipeline, versioniert — Änderungen werden reviewt

---

## Schnellstart (Entwicklung)

Voraussetzungen: Go, Node, Docker, `jq`, `git`.

Nach dem Klonen einmalig — aktiviert Hooks und Commit-Vorlage:

    git clone <repo-url> && cd healthgate
    tools/setup.sh
    cp .env.example .env

Backend und Tests:

    make test                     # Unit-Tests mit Coverage
    make lint                     # go vet und gofmt
    make dev                      # Backend auf :8080
    curl localhost:8080/healthz
    curl localhost:8080/metrics

Frontend getrennt, proxyt `/api` auf das Backend:

    cd web && npm install && npm run dev      # :5173

E2E-Tests lokal:

    cd tests/e2e && npm install && npx playwright install chromium
    BASIS_URL=http://localhost:5173 npx playwright test

Umgebungen:

    make staging-up               # Staging auf :8081
    make prod-up                  # beide Slots plus Caddy auf :80
    make monitoring-up            # Prometheus :9090, Grafana :3000

**Merksatz:** `make monitoring-up` setzt voraus, dass Produktion läuft — das
Docker-Netz `healthgate` wird von `docker-compose.prod.yml` angelegt und vom
Monitoring-Stack als extern eingebunden.

Ohne `.env` startet nichts: die Compose-Dateien lesen Datenbankwerte daraus. Eine
Anmeldung existiert nicht, weder in Entwicklung noch in Produktion — siehe
[Sicherheit](#sicherheit).

---

## Konfiguration

Ausschließlich Umgebungsvariablen. Pflichtwerte werden beim Start geprüft und
führen bei Fehlen oder Unsinn zum sofortigen Abbruch, nicht zu stillen
Standardwerten. `.env.example` enthält neutrale Platzhalter; `.env` ist in
`.gitignore` und gehört niemals ins Repository.

### Anwendung

| Variable | Pflicht | Beschreibung |
|---|---|---|
| `HEALTHGATE_ADRESSE` | nein | Adresse des HTTP-Servers, Vorgabe `:8080` |
| `HEALTHGATE_UMGEBUNG` | nein | `lokal`, `staging` oder `prod`; steuert die Pflichtprüfungen |
| `HEALTHGATE_SLOT` | in `prod` | `blue` oder `green`; ohne gültigen Wert bricht der Start ab |
| `HEALTHGATE_VERSION` | in `prod` | Git-SHA, wird von der Pipeline gesetzt; `dev` ist in `prod` unzulässig |
| `HEALTHGATE_CHAOS_RATE` | nein | Anteil absichtlicher 500er, `0.0` bis `1.0`, Vorgabe `0.0` |
| `HEALTHGATE_WEB_VERZEICHNIS` | nein | Pfad der Frontend-Dateien, Vorgabe `/srv/web` |
| `HEALTHGATE_DB_URL` | in `prod` | Verbindung zu PostgreSQL; ohne sie läuft der In-Memory-Speicher |

### Datenbank und Deployment

| Variable | Pflicht | Beschreibung |
|---|---|---|
| `POSTGRES_DB` | ja | Datenbankname, von beiden Compose-Dateien gelesen |
| `POSTGRES_USER` | ja | Datenbankbenutzer |
| `POSTGRES_PASSWORD` | ja | Kennwort |
| `VERSION_BLUE` | nein | ausgelieferte Version in Slot blau, Vorgabe `dev` |
| `VERSION_GREEN` | nein | ausgelieferte Version in Slot grün, Vorgabe `dev` |
| `CHAOS_BLUE` | nein | Chaos-Rate für Slot blau, Vorgabe `0.0` |
| `CHAOS_GREEN` | nein | Chaos-Rate für Slot grün, Vorgabe `0.0` |
| `HEALTHGATE_IMAGE` | ja | auszulieferndes Image, z. B. `healthgate:abc1234`; von der Pipeline gesetzt |
| `REGISTRY` | ja für Push | Ziel der Container-Images |
| `CADDY_CONTAINER` | nein | Containername für den Neuladevorgang, Vorgabe `healthgate-prod-caddy-1` |
| `KONF_DATEI` | nein | Pfad zu `active-slot.conf`, überschreibt die Vorgabe |
| `ZUSTAND_VERZEICHNIS` | nein | Pfad zu `deploy/state` |
| `BASIS_URL` | nein | Basis-URL für Playwright und `rollback.sh` |
| `STAGING_PORT` | nein | Host-Port von Staging, Vorgabe `8081`; `0` vergibt einen freien Port |
| `WAIT_PAUSE` | nein | Sekunden zwischen Health-Abfragen, Vorgabe `2` |

### Health-Gate und Monitoring

| Variable | Pflicht | Beschreibung |
|---|---|---|
| `PROMETHEUS_URL` | ja für `observe.sh` | Adresse der Prometheus-HTTP-API |
| `OBSERVE_DAUER` | nein | Länge des Beobachtungsfensters in Sekunden, Vorgabe `120` |
| `OBSERVE_INTERVALL` | nein | Abstand der Abfragen in Sekunden, Vorgabe `10` |
| `OBSERVE_SCHWELLE` | nein | zulässiger 5xx-Anteil, Vorgabe `0.05` |
| `OBSERVE_MIN_ANFRAGEN` | nein | Mindestlast, unter der das Fenster als aussagelos gemeldet wird, Vorgabe `5` |
| `GRAFANA_ADMIN_PASSWORT` | ja | Kennwort des Grafana-Administrators |

**Merksatz:** `OBSERVE_MIN_ANFRAGEN` ist keine Kosmetik. Ohne Verkehr sagt ein
grünes Fenster nichts aus; `observe.sh` weist darauf hin, statt ein Deployment
stillschweigend zu bestätigen.

### Dashboard

`monitoring/grafana/provisioning/dashboards/healthgate.json` wird beim Start von
Grafana geladen und liegt im Ordner *healthgate*. Es ist eine Datei im
Repository und kein in der Oberfläche geklicktes Dashboard: sonst lebte es im
Grafana-Volume, wäre nicht reviewbar und beim nächsten frischen Aufsetzen weg
(Entscheidung E-019).

| Panel | Zeigt |
|---|---|
| Fehlerrate je Slot | denselben Ausdruck, über den `observe.sh` entscheidet, mit Grenzwertlinie bei 5 % |
| Slot mit Nutzerverkehr | welcher Slot bedient; Health-Checks sind ausgenommen |
| Anfragen je Sekunde je Slot | das Umschalten als Übergang, einschließlich Health-Checks |
| Mittlere Antwortzeit je Route | Summe durch Anzahl; Quantile gibt es bewusst nicht (E-004) |
| Ausgelieferte Version je Slot | was gerade wo läuft |
| Laufzeit je Slot | ein Sprung auf null ist ein Neustart |

Zwei Deployment-Marker als Annotation, beide aus den Metriken abgeleitet und
nicht von der Pipeline gesetzt: **Deployment** markiert den Neustart eines Slots,
**Umschalten** den Moment, in dem ein Slot Verkehr bekommt. Der Weg über die
Grafana-API bräuchte ein Token in den Jenkins-Credentials und setzte den Marker
auch dann, wenn das Deployment danach zurückgerollt wird (Entscheidung E-020).

Grafana liest Dashboards alle 30 Sekunden neu ein, Datenquellen dagegen nur beim
Start. Nach einer Änderung an `provisioning/datasources/` gehört deshalb ein

    docker restart healthgate-monitoring-grafana-1

dazu — und ein Blick, ob der Container danach auch oben bleibt (E-025).

**Merksatz:** Caddy prüft nur den Slot, der Verkehr bekommt. Deshalb ist das
Umschalten im Diagramm auch dann zu sehen, wenn niemand die Anwendung benutzt.

---

## Betriebsskripte

    deploy/scripts/active-slot.sh                  Slot ausgeben, der Verkehr bekommt
    deploy/scripts/target-slot.sh                  den untätigen Slot ausgeben
    deploy/scripts/wait-healthy.sh <ziel> [n] [max] warten, bis n Abfragen in Folge grün sind
                                                   ziel: URL oder container:<name>
    deploy/scripts/switch-slot.sh <blue|green>     Verkehr umschalten, vorherigen Slot vermerken
    deploy/scripts/observe.sh <blue|green>         Beobachtungsfenster; Exitcode 1 bei Verstoß
    deploy/scripts/rollback.sh [grund]             auf den vermerkten Slot zurückschalten
    deploy/scripts/last-erzeugen.sh [url] [rate]   Grundlast für die Vorführung

    tools/setup.sh                                 nach dem Klonen: Hooks und Vorlagen
    tools/story-start.sh <story-id> "<titel>"      Branch für eine Story anlegen
    tools/story-pr.sh                              pushen und Pull Request öffnen
    tools/release.sh <major|minor|patch>           Changelog, Tag, Release-PR

Verkürzt über `make`: `switch SLOT=green`, `observe SLOT=green`, `rollback`.

---

## API

Keine Authentifizierung. Fehler werden einheitlich als
`{"fehler": "<Beschreibung>"}` mit passendem Statuscode geliefert.

| Methode | Pfad | Zweck | Antwort |
|---|---|---|---|
| `GET` | `/healthz` | Bereitschaft | `200` mit Status, Version, Slot; `503` mit Grund |
| `GET` | `/metrics` | Prometheus-Textformat | `200` |
| `POST` | `/api/links` | Kurzlink anlegen | `201` mit Link; `400` bei ungültigem Ziel; `409` bei belegtem Slug |
| `GET` | `/api/links` | Liste, neueste zuerst | `200` mit `{"links": []}` |
| `DELETE` | `/api/links/{slug}` | Kurzlink entfernen | `204`; `404` wenn unbekannt |
| `GET` | `/{slug}` | Weiterleitung | `302` mit `Location`; sonst Frontend bzw. `404` |
| `GET` | `/` | Frontend | statische Dateien |

**Merksatz:** Die Weiterleitung antwortet mit `302`, nicht `301`. Eine dauerhafte
Weiterleitung wird vom Browser gecacht — beim Wiederholen der E2E-Tests und beim
Zählen der Aufrufe wäre das eine stille Fehlerquelle.

---

## Tests und Qualitätssicherung

    make lint                     # go vet, gofmt
    make test                     # Unit-Tests mit Coverage
    make e2e                      # Playwright gegen BASIS_URL

Vor jedem Commit prüfen die Hooks Formatierung, `go vet`, gestagte `.env`-Dateien
und eine lokale Sperrliste. Vor jedem Push laufen die Unit-Tests und ein direkter
Push auf `main` wird abgelehnt. Siehe `docs/git-konventionen.md`.

### Coverage und Testberichte

Die Pipeline verlangt **mindestens 65 Prozent** Statement-Coverage über alle
Pakete. Der Wert liegt unter dem aktuellen Stand von rund 72 Prozent: er soll
einen Einbruch melden, nicht jede Nachkommastelle (Entscheidung E-017).

Die Tests des PostgreSQL-Speichers laufen gegen eine echte Datenbank. Die
Pipeline startet dafür einen Wegwerf-Container und setzt `HEALTHGATE_TEST_DB_URL`;
ohne die Variable überspringen sich diese Tests. Lokal:

    docker run -d --name hg-test-db -P \
      -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=test postgres:16-alpine
    PORT=$(docker port hg-test-db 5432/tcp | head -1 | sed 's/.*://')
    cd app && HEALTHGATE_TEST_DB_URL="postgres://test:test@127.0.0.1:$PORT/test?sslmode=disable" \
      go test ./... -cover

**Merksatz:** Übersprungene Tests sind schlimmer als fehlende. Ohne Datenbank
weist die Coverage-Zahl eine Prüfung aus, die nicht stattgefunden hat — deshalb
gehört die Testdatenbank in die Pipeline (Entscheidung E-018).

`go-junit-report` wandelt die Ausgabe von `go test` in einen JUnit-Bericht; die
Playwright-Suite schreibt ihren eigenen. Jenkins zeigt beides als Tabelle mit
Verlauf statt als Textwand im Konsolenprotokoll.

### Worauf die Tests besonders achten

**`/healthz` bleibt auch bei Chaos-Rate `1.0` grün.** Wäre der Health-Check
mitbetroffen, bräche die Pipeline vor dem Umschalten ab — der metrikbasierte
Rollback, also das Kernmerkmal des Projekts, würde nie erreicht.

**`/metrics` wird nicht mitgezählt.** Prometheus fragt den Endpunkt im
Sekundentakt ab. Würde die Middleware ihn zählen, verfälschten die eigenen
Abfragen die Statistik, auf deren Grundlage der Health-Gate entscheidet.

**Pfade werden auf eine feste Menge von Bezeichnern abgebildet.** Ein Test prüft
alle Fälle einzeln, weil jeder Slug als eigenes Label eine eigene Zeitreihe
erzeugen würde — Prometheus wäre binnen Tagen unbenutzbar.

**Die Weiterleitung liefert `302`.** Explizit geprüft, weil ein versehentliches
`301` erst Wochen später als scheinbar zufällig falsche Aufrufzahlen auffällt.

**Reservierte Slugs werden abgelehnt, auch in Großschreibung.** Ohne diese Regel
könnte ein Wunsch-Slug `api` oder `healthz` eigene Pfade verdecken.

**Ein doppelter Wunsch-Slug ergibt `409`, nicht `500`.** Der Test legt denselben
Slug zweimal an, weil die Unterscheidung zwischen Konflikt und Serverfehler
darüber entscheidet, ob das Frontend eine brauchbare Meldung zeigen kann.

**Produktion ohne gültigen Slot bricht beim Start ab.** Ohne Slot sind die
Metriken nicht zuordenbar und der Health-Gate wertet ins Leere. Der Test setzt
`HEALTHGATE_UMGEBUNG=prod` mit ungültigem Slot und erwartet einen Fehler.

**Eine Chaos-Rate außerhalb `0` bis `1` bricht ab.** Ein Tippfehler in der
Deployment-Variable soll auffallen, bevor Verkehr darauf läuft.

**Erzeugte Slugs streuen.** Fünfzig Läufe müssen mindestens 45 verschiedene
Ergebnisse liefern. Ein defekter Zufallsgenerator fällt sonst erst beim ersten
Kollisionsfehler in Produktion auf.

---

## CI/CD und Deployment

Die Pipeline liegt als `Jenkinsfile` im Repository, damit eine Änderung an der
Auslieferung genauso reviewt wird wie eine Änderung am Code.

| Stage | Prüft bzw. tut | Arbeitet in |
|---|---|---|
| Vorbereitung | Werkzeuge vorhanden, `.env` lesbar, Build-Name auf Git-SHA setzen | Workspace |
| Statische Analyse | `go vet`; unformatierter Code bricht ab | Workspace |
| Unit-Tests | Tests gegen eine Wegwerf-Datenbank, Coverage-Schwelle, JUnit-Bericht | Workspace |
| Image bauen | ein Image mit SHA-Tag, lokal und für die Registry | Workspace |
| Image veröffentlichen | Push in die Registry (offen, Story C-05) | Workspace |
| Staging ausliefern | Staging mit genau diesem Image in einem Stack je Branch, auf Bereitschaft warten | Workspace |
| E2E-Tests gegen Staging | Playwright; Bericht und Spuren als Artefakt | Workspace |
| Deployment-Verzeichnis prüfen | Schreibrecht und Stand des Deployments, nur auf `main` | Deployment |
| Freigabe für Produktion | bewusste menschliche Entscheidung, nur auf `main` | — |
| Deployment-Verzeichnis aktualisieren | Checkout des gebauten Commits aus dem Workspace; aktiver Slot bleibt unberührt | Deployment |
| Reverse Proxy abgleichen | Caddy auf den geprüften Stand, ohne die Slots mitzuziehen | Deployment |
| Zielslot bespielen | untätigen Slot mit dem gebauten Image starten | Deployment |
| Prüfung vor dem Umschalten | dreimal `/healthz` direkt am Container | Deployment |
| Umschalten | Caddy auf den neuen Slot, vorherigen vermerken | Deployment |
| Beobachtungsfenster | 5xx-Anteil gegen den Grenzwert; Exitcode steuert den Rollback | Deployment |

### Workspace und Deployment-Verzeichnis

Gebaut und getestet wird im Jenkins-Workspace, ausgeliefert wird gegen
`/home/admin/healthgate`. Der Workspace ist bei jedem Lauf ein frischer Checkout
und kennt den laufenden Zustand nicht; Caddy mountet `active-slot.conf` aus dem
Deployment-Verzeichnis. Liefe `switch-slot.sh` im Workspace, beschriebe es die
Workspace-Kopie: das Umschalten bliebe wirkungslos, `observe.sh` beobachtete den
falschen Slot, und der Build wäre grün, ohne dass etwas passiert ist
(Entscheidung E-012).

Voraussetzungen auf der Maschine, einmalig einzurichten:

| Was | Warum |
|---|---|
| `/etc/healthgate/.env`, Gruppe `jenkins`, Rechte `640` | Zugangsdaten gehören nicht in den Workspace (E-014) |
| Jenkins-Benutzer mit Schreibrecht auf `/home/admin/healthgate` | Deploy-Stages schreiben den aktiven Slot und die Historie |
| setgid auf `deploy/caddy` und `deploy/state` | Laufzeitdateien bleiben für beide Schreiber les- und schreibbar (E-024) |

Das Schreibrecht wird über die Gruppe erteilt, nicht über `sudo`:

    sudo usermod -aG admin jenkins
    sudo systemctl restart jenkins
    chmod g+s deploy/caddy deploy/state

Die Stage `Deployment-Verzeichnis prüfen` bricht mit genau diesem Hinweis ab,
wenn das Recht fehlt — und zwar vor der Freigabe, nicht mitten im Umschalten.

**Merksatz:** Das setgid-Bit ist kein Detail. `active-slot.conf` wird
abwechselnd von der Pipeline und von Hand geschrieben; ohne gemeinsame Gruppe
legt der eine sie mit `600` an und der andere kommt nicht mehr heran. Caddy
liest sie als root im Container weiter — von aussen sieht dann alles gesund aus,
und nur der Rollback von Hand ist still kaputt (E-024).

Weil das Verzeichnis dem Benutzer `admin` gehört und nicht Jenkins, verweigert
git dort sonst jede Operation. Die Ausnahme steht als `safe.directory` im
`environment`-Block der betroffenen Stages und nicht in der gitconfig des
Agenten — sonst hinge die Pipeline an Zustand, den niemand versioniert (E-021).

Zugangsdaten für GitHub braucht das Deployment-Verzeichnis keine: es holt den
Stand aus dem Jenkins-Workspace, der bereits auf dem gebauten Commit steht. Das
ist nicht nur bequemer, sondern richtiger — ausgeliefert werden soll der Stand,
der gerade geprüft wurde, und nicht der, der inzwischen auf `main` liegt (E-023).

**Merksatz:** Beim ersten Deployment nach dieser Umstellung entfernt der
Checkout im Deployment-Verzeichnis die dort noch versionierte
`active-slot.conf`. Die Stage sichert sie vorher und stellt sie danach wieder
her. Ohne das käme sie aus der Vorlage zurück — und die zeigt auf `blue`.

### Deploy-Ablauf

Der untätige Slot wird bespielt, während der aktive weiterläuft. Erst nach
mehreren erfolgreichen Health-Abfragen am neuen Container schaltet Caddy den
Verkehr um; der vorherige Slot wird dabei in `deploy/state/vorheriger-slot`
vermerkt. Danach beobachtet die Pipeline die Fehlerrate. Bei Überschreitung ruft
der `post`-Block `rollback.sh` auf: der Verkehr geht zurück, der Build wird als
fehlgeschlagen markiert und der Grund benannt. Ein Rollback ohne
vorangegangenes Umschalten findet nicht statt — das steuert das Merkmal
`UMGESCHALTET`.

### Supply Chain

Images werden über den Git-SHA eindeutig referenziert. Gebaut wird genau einmal
je Commit; Staging und beide Produktionsslots ziehen dasselbe Artefakt über
`HEALTHGATE_IMAGE`. Baute Produktion neu, wäre das ausgelieferte Image nicht das
getestete (Entscheidung E-013). Zugangsdaten liegen ausschließlich in den
Jenkins-Credentials und erscheinen nicht im Build-Log. Das Laufzeitbild führt
die Anwendung als unprivilegierter Benutzer aus, CGO ist abgeschaltet.

Offen: SHA-Pinning der Pipeline-Werkzeuge und eine Dependabot-Politik. Da die
Pipeline auf Jenkins läuft, greift die übliche GitHub-Actions-Automatisierung
hier nicht — die Abhängigkeiten von `app/`, `web/` und `tests/e2e/` sind noch
manuell zu pflegen.

---

## Sicherheit

- Keine Zugangsdaten im Repository; Konfiguration ausschließlich über
  Umgebungsvariablen mit Fail-fast bei fehlenden Pflichtwerten.
- `pre-commit`-Hook lehnt gestagte `.env`-Dateien ab und prüft gegen eine lokale,
  nicht versionierte Sperrliste mit echten Domains und Namen.
- `/metrics` ist am Reverse Proxy nach außen gesperrt; Prometheus greift die Slots
  im internen Netz ab.
- Ziel-URLs werden auf `http` und `https` mit vorhandenem Host geprüft, wodurch
  `javascript:`-Ziele abgelehnt werden.
- Der Anfragekörper beim Anlegen ist auf 8 KiB begrenzt.
- Erzeugte Slugs stammen aus `crypto/rand` und sind nicht erratbar.
- Die Anwendung läuft im Container als unprivilegierter Benutzer.
- Der Chaos-Schalter hat keinen Endpunkt; die Rate ist eine Eigenschaft des
  Deployments.

**Die Oberfläche ist keine Zugriffskontrolle.** Es gibt hier bewusst gar keine:
Authentifizierung und Rollen stehen ausdrücklich nicht im Projektumfang, weil der
Schwerpunkt die Auslieferungsstrecke ist. Wer diese Anwendung öffentlich
betreibt, kann von jedem Kurzlinks anlegen und löschen lassen. Das ist eine
Scope-Entscheidung, keine Nachlässigkeit — aber sie muss beim Betrieb bekannt
sein.

Betriebs- und Infrastrukturdokumentation liegt bewusst nicht im Repository.

---

## Umsetzungsstand

### Fertig und im Einsatz

- Anwendung mit Anlegen, Auflösen, Auflisten, Löschen und Wunsch-Slug
- `/healthz` und `/metrics` im Prometheus-Textformat, ohne externe Abhängigkeit
- Chaos-Schalter über Umgebungsvariable, `/healthz` ausgenommen
- Unit-Tests über Fachlogik, Konfiguration, Metriken und Handler
- Umschalt-, Warte-, Beobachtungs- und Rollback-Skripte
- Compose-Dateien für Staging und Produktion mit beiden Slots
- Prometheus mit Alarmregeln, Grafana mit Datenquelle und provisioniertem Dashboard
- Playwright-Suite über den vollständigen Ablauf
- `Jenkinsfile` mit allen Stages einschließlich Rollback im `post`-Block
- Git-Arbeitsablauf: Hooks, Vorlagen, Story- und Release-Werkzeuge

### Wartet auf Daten oder Entscheidungen außerhalb

- Branch-Schutz auf `main` — setzt Administratorrechte am Repository voraus
- Kontextname der Jenkins-Statusprüfung — erst nach dem ersten Build bekannt
- Registry-Adresse und Zugangsdaten für den Image-Push (Story C-05, C-08)
- Jenkins-Installation mit Docker-fähigem Agenten (Story C-02, C-04)

### Für spätere Iterationen vorgesehen

- Migrationswerkzeug in der Pipeline statt SQL beim Datenbankstart (siehe E-010)
- Alertmanager mit echter Benachrichtigung (O-06)
- Deployment-Historie als Ansicht statt als TSV-Datei
