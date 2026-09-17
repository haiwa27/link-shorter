# Übergabe

Stand: 17.09.2026. Die Strecke läuft von Ende zu Ende. Build #7 auf `main` ist
vollständig durchgelaufen: gebaut, gegen Staging geprüft, freigegeben, Slot
`blue` bespielt, umgeschaltet, 120 Sekunden beobachtet -- Fehleranteil 0,0000
bei 7 bis 20 Anfragen pro Minute, kein Rollback nötig.

**Produktion läuft jetzt auf Slot `blue` mit Version `c43ef51`**, vorher grün
mit `fdbce28`. `deploy/state/vorheriger-slot` steht auf `green`, das Rollback-
Ziel ist also gesetzt und der alte Slot läuft weiter. An `/etc/healthgate/.env`,
`/home/admin/healthgate/.env`, an Volumes und an der Cloudflare-Konfiguration
wurde nichts geändert.

## Pull Requests

| PR | Story | Inhalt | Stand |
|---|---|---|---|
| [#10](https://github.com/haiwa27/healthgate/pull/10) | C-02, C-04 | Pipeline bis einschliesslich E2E, Deploy gegen `/home/admin/healthgate` | **gemerged** |
| [#14](https://github.com/haiwa27/healthgate/pull/14) | C-04 | `safe.directory`, Rettung der `active-slot.conf`, Staging je Branch | **gemerged** |
| [#9](https://github.com/haiwa27/healthgate/pull/9) | P-01 | PostgreSQL als Speicher hinter `store.Speicher` | **gemerged** |
| [#11](https://github.com/haiwa27/healthgate/pull/11) | Q-03, Q-06 | Coverage-Schwelle 65 Prozent, Testdatenbank, JUnit-Berichte | **gemerged** |
| [#12](https://github.com/haiwa27/healthgate/pull/12) | O-04, O-05 | Grafana-Dashboard als Provisioning-Datei, Deployment-Marker | **gemerged** |
| [#13](https://github.com/haiwa27/healthgate/pull/13) | D-01 | diese Datei | **gemerged** |
| [#15](https://github.com/haiwa27/healthgate/pull/15) | C-04 | Deployment-Stand aus dem Workspace statt von GitHub | **gemerged** |
| [#16](https://github.com/haiwa27/healthgate/pull/16) | C-04 | Rechte der Slot-Datei für beide Schreiber erhalten | offen |

### Was der erste echte Deploy zutage gefördert hat

Die Deploy-Stages waren bis dahin geschrieben, aber nie gelaufen. Drei Dinge
fielen erst im laufenden Betrieb auf, jedes davon still genug, um es beinahe zu
übersehen:

1. **`dubious ownership`** -- git verweigert die Arbeit in einem Verzeichnis,
   das einem anderen Benutzer gehört (#14, E-021).
2. **`could not read Username for 'https://github.com'`** -- der Jenkins-Benutzer
   hat keine Anmeldung für das private Repository. Behoben, indem das
   Deployment-Verzeichnis den Stand aus dem Workspace holt statt von GitHub --
   was ohnehin richtiger ist, weil dort genau der geprüfte Commit liegt
   (#15, E-023).
3. **`active-slot.conf` mit `600` und Eigentümer `jenkins`** -- die Pipeline legte
   sie aus einer `mktemp`-Kopie neu an. Danach scheiterte jeder Handgriff an
   `Permission denied`, während Caddy die Datei als root im Container
   weiterlas: von aussen sah alles gesund aus, nur der Rollback von Hand war
   still kaputt. Rechte auf der Maschine repariert, Wiederholung verhindert
   (#16, E-024).

### Wie die Konflikte aufgelöst wurden

Kollidiert ist immer dasselbe: jeder Branch hängt seine Entscheidungen ans Ende
von `docs/entscheidungen.md` und streicht in der README einen Punkt aus der Liste
der offenen Themen.

- `docs/entscheidungen.md`: beide Seiten behalten, Einträge nach Nummer sortiert
  (E-008 bis E-024).
- `README.md`: unter *Für spätere Iterationen vorgesehen* fällt jeder Punkt weg,
  dessen Story gemerged ist -- nicht der eine oder der andere, sondern beide.
- `Jenkinsfile` (#11): der `environment`-Block enthält beides -- die Grenzwerte
  des Health-Gates aus #10 und `COVERAGE_SCHWELLE`, `JUNIT_REPORT_VERSION` und
  `TEST_DB` aus #11.
- `.gitignore` (#11): beide Blöcke, Testberichte und aktiver Slot.

## Was erledigt ist

### P-01: PostgreSQL als Speicher (PR #9)

`store.Speicher` hat eine zweite Implementierung auf Basis von `pgx/v5`. Die
Schnittstelle ist unverändert, Handler und bestehende Tests sind nicht angefasst.
`Pruefen` macht einen echten Ping; ein Slot ohne Datenbank meldet damit 503 statt
sich als bereit auszugeben. Der Aufrufzähler wird in der Datenbank erhöht, sonst
hinge die Zahl daran, welcher Slot die Weiterleitung bedient hat. In `prod`
erzwingt `config.Laden` jetzt `HEALTHGATE_DB_URL`.

15 Tests ergänzt. Die Tests gegen eine echte Datenbank laufen, wenn
`HEALTHGATE_TEST_DB_URL` gesetzt ist, und überspringen sich sonst. Sie wurden
gegen PostgreSQL 16 auf der Maschine ausgeführt: alle grün, Coverage des Pakets
`store` 92,1 Prozent.

### C-02 und C-04: Pipeline (PR #10)

Die Pipeline läuft von oben bis einschliesslich der E2E-Tests durch. Nachweis:
`healthgate/PR-10`, Build 2, SUCCESS, vier Playwright-Tests grün.

Behoben bzw. geändert:

- `.env` wird aus `/etc/healthgate/.env` gelesen statt aus dem Workspace.
- Alle Deploy-Stages arbeiten gegen `/home/admin/healthgate`. Die Regel steht im
  Kopf des `Jenkinsfile` und als Entscheidung E-012.
- Das Image wird einmal gebaut; Staging und beide Produktionsslots ziehen genau
  dieses Artefakt über `HEALTHGATE_IMAGE` (E-013). Vorher baute Produktion neu
  und lieferte damit etwas anderes aus als das, was getestet wurde.
- `deploy/caddy/active-slot.conf` ist nicht mehr versioniert. Im Repository liegt
  `active-slot.conf.vorlage`; `switch-slot.sh` legt die Datei beim ersten Lauf an
  (E-015). Versioniert setzte jeder Checkout im Deployment-Verzeichnis den
  aktiven Slot still zurück.
- Caddy mountet das Verzeichnis `deploy/caddy` statt der Einzeldateien. Ein
  Mount auf eine Einzeldatei hängt an deren Inode; wird die Datei ersetzt statt
  überschrieben, sieht der Container weiter den alten Inhalt.
- `wait-healthy.sh` kann jetzt auch `container:<name>` prüfen. Die
  Produktionsslots veröffentlichen keinen Port auf dem Host, die bisherige
  Prüfung vor dem Umschalten konnte deshalb nie funktionieren.
- Playwright läuft im mitgelieferten Container: die Maschine hat Node 18,
  Playwright verlangt Node 20 (E-016).

### Q-03 und Q-06: Coverage und Testberichte (PR #11)

Die Schwelle steht bei 65 Prozent statt 40. 40 lag unter dem tatsächlichen Stand
und hätte einen Einbruch nie gemeldet. Gemessen mit laufender Testdatenbank
liegt der Stand bei 72,4 Prozent, wenn #9 und #11 beide gemerged sind.

Die Stage startet vor den Tests einen PostgreSQL-Container und räumt ihn im
`post`-Block ab. Ohne ihn überspringen sich die Datenbanktests und die
Coverage-Zahl wiese eine Prüfung aus, die nicht stattgefunden hat (E-018).

`go-junit-report` und der JUnit-Bericht von Playwright werden über den
`junit`-Schritt ausgewertet. Nebenbei behoben: `go test ... | tee` lieferte den
Exitcode von `tee`, fehlgeschlagene Tests hätten den Build nicht rot gemacht.

Tests für den In-Memory-Speicher ergänzt, der bisher ohne jede Prüfung lief.

### O-04 und O-05: Dashboard und Marker (PR #12)

`monitoring/grafana/provisioning/dashboards/healthgate.json` mit sechs Panels,
darunter die Fehlerrate mit dem Ausdruck, über den `observe.sh` entscheidet, und
die Anfragerate je Slot, in der das Umschalten als Übergang zu sehen ist.

Zwei Deployment-Marker als Annotation, beide aus den Metriken abgeleitet statt
über die Grafana-API gesetzt (E-020). Die Datenquelle bekommt eine feste `uid`,
sonst zeigt das Dashboard nach einer Neuinstallation ins Leere.

Geprüft in einem Wegwerf-Grafana auf Port 3001 gegen den echten Prometheus:
Dashboard provisioniert, sechs Panels, zwei Annotationen, alle Abfragen liefern
Daten. Der Testcontainer ist wieder entfernt, das laufende Grafana wurde nicht
angefasst.

## Was nicht funktioniert hat

Die drei Fehlschläge aus dem ersten echten Deploy stehen oben; alle drei sind
behoben, der letzte davon in #16 und damit noch offen. Was währenddessen an
Zeit verloren ging, ging für dieselbe Sorte Fehler drauf: Dinge, die im Trockenen
nicht auffallen, weil sie erst entstehen, wenn ein zweiter Benutzer dieselben
Dateien anfasst.

Nicht behoben, weil es eine Entscheidung von euch braucht:

- **Die Freigabe blockiert 15 Minuten lang einen Executor.** Jeder Merge löst
  einen Build aus, der bei *Freigabe für Produktion* wartet. Wer mehrere PRs
  hintereinander merged, sollte die Zwischenläufe abbrechen und nur den letzten
  freigeben, sonst stehen sie sich gegenseitig im Weg.
- **Je Branch bleibt ein Staging-Volumen zurück.** Der Preis für die Isolation
  aus E-022. Bei vielen kurzlebigen Branches lohnt ein `docker volume prune`
  von Zeit zu Zeit.

### Weiter offen, bewusst nicht angefasst

- **C-05, C-08:** eigene Registry und Push. Solange Jenkins und Produktion auf
  derselben Maschine liegen, reicht der lokale Docker-Daemon. Der Push braucht
  Zugangsdaten, die es noch nicht gibt.
- **Migrationswerkzeug:** `0001_init.sql` wird über
  `docker-entrypoint-initdb.d` angewendet. Das läuft nur bei leerem Volume; eine
  spätere Migration 0002 käme auf einer bestehenden Datenbank nicht an. Bekannt
  und in E-010 begründet, mit dem Auslöser: sobald die zweite Migration ansteht,
  kommt golang-migrate als eigene Stage vor dem Umschalten.
- **O-06:** Alertmanager mit echter Benachrichtigung.

## Entscheidungen dieser Sitzung

Alle in `docs/entscheidungen.md` mit Alternativen und Begründung:

| ID | Kurz |
|---|---|
| E-008 | pgx statt `database/sql` |
| E-009 | Verbindungspool ohne Verbindungsaufbau beim Start |
| E-010 | Schema über die Migration beim Datenbankstart, kein Migrationswerkzeug |
| E-011 | Datenbanktests gegen eine echte Wegwerf-Instanz statt gegen Attrappen |
| E-012 | Bauen im Workspace, Ausliefern gegen das Deployment-Verzeichnis |
| E-013 | Ein Image bauen und dasselbe ausliefern |
| E-014 | `.env` ausserhalb des Workspace unter `/etc/healthgate` |
| E-015 | Der aktive Slot ist Laufzeitzustand und wird nicht versioniert |
| E-016 | Playwright im mitgelieferten Container |
| E-017 | Coverage-Schwelle bei 65 Prozent |
| E-018 | Testdatenbank in der Pipeline |
| E-019 | Dashboard als Provisioning-Datei |
| E-020 | Deployment-Marker aus den Metriken statt über die Grafana-API |
| E-021 | `safe.directory` als Variable der Stage statt in der gitconfig des Agenten |
| E-022 | Eigener Staging-Stack je Branch statt eines gemeinsamen |
| E-023 | Deployment-Stand aus dem Workspace statt von GitHub |
| E-024 | Die Slot-Datei gehört beiden Schreibern, nicht dem zuletzt Schreibenden |

## Was als Nächstes zu tun ist

1. **#16 prüfen und mergen.** Der PR gehört der jeweils anderen Person zum
   Review -- alle hier stammen aus derselben Sitzung und haben noch niemanden
   gesehen.
2. **Das Monitoring einmal neu laden**, damit Grafana das Dashboard aus #12
   einliest:

       docker compose -f monitoring/docker-compose.monitoring.yml --env-file .env up -d grafana

3. **Den Rollback vorführen.** Das ist der einzige Teil des Ablaufs, der noch
   nie im Ernst gelaufen ist: das Health-Gate hat bisher nur bestätigt, nie
   abgebrochen. Dafür den Chaos-Wert des Zielslots setzen -- der nächste
   Zielslot ist `green`, also `CHAOS_GREEN=0.3` in `/etc/healthgate/.env` --
   und einen Build laufen lassen. Das Beobachtungsfenster muss dann verletzt
   melden, `rollback.sh` auf `blue` zurückschwenken und der Build rot bleiben.
4. **Dabei Last erzeugen.** Ohne Verkehr ist das Fenster aussagelos, und
   `observe.sh` sagt das auch. In einem zweiten Terminal:

       deploy/scripts/last-erzeugen.sh http://localhost 5

   Im Durchlauf von Build #7 kamen so 7 bis 20 Anfragen pro Minute zusammen --
   genug, damit die Zahlen etwas bedeuten.
5. **Nach der Vorführung `CHAOS_GREEN` wieder auf `0.0` setzen.** Sonst
   scheitert das nächste echte Deployment an einem Fehler, den jemand absichtlich
   eingebaut und vergessen hat.
