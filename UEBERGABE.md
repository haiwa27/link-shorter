# Übergabe

Stand: 17.09.2026. Fünf Stories umgesetzt, jede als eigener Branch mit eigenem
Pull Request. **#10 ist gemerged**, die übrigen warten auf euer Review -- die
Freigabe liegt bei euch.

Produktion lief während der gesamten Arbeit durch: Slot grün, Version `fdbce28`,
`curl localhost/healthz` nach jedem Eingriff geprüft. An `/etc/healthgate/.env`,
`/home/admin/healthgate/.env`, an Volumes und an der Cloudflare-Konfiguration
wurde nichts geändert.

## Pull Requests

| PR | Story | Inhalt | Build |
|---|---|---|---|
| [#9](https://github.com/haiwa27/healthgate/pull/9) | P-01 | PostgreSQL als Speicher hinter `store.Speicher` | grün (Build 2) |
| [#10](https://github.com/haiwa27/healthgate/pull/10) | C-02, C-04 | Pipeline bis einschliesslich E2E, Deploy gegen `/home/admin/healthgate` | **gemerged** |
| [#11](https://github.com/haiwa27/healthgate/pull/11) | Q-03, Q-06 | Coverage-Schwelle 65 Prozent, Testdatenbank, JUnit-Berichte | grün (Build 3) |
| [#12](https://github.com/haiwa27/healthgate/pull/12) | O-04, O-05 | Grafana-Dashboard als Provisioning-Datei, Deployment-Marker | grün (Build 3) |
| [#13](https://github.com/haiwa27/healthgate/pull/13) | D-01 | diese Datei | grün (Build 3) |
| [#14](https://github.com/haiwa27/healthgate/pull/14) | C-04 | `safe.directory`, Rettung der `active-slot.conf`, Staging je Branch | grün (Build 2) |

In alle offenen Branches ist `main` nach dem Merge von #10 hineingezogen worden,
die Konflikte sind aufgelöst, alle fünf Builds sind grün.

**#14 zuerst mergen**: `main` scheitert derzeit an der Stage
*Deployment-Verzeichnis prüfen*, und genau das behebt #14. Danach #9, dann #11,
dann #12, dann #13. #11 setzt die Coverage-Schwelle auf einen Wert, den erst die
Tests aus #9 vollständig tragen.

### Wie die Konflikte aufgelöst wurden

- `docs/entscheidungen.md`: beide Seiten behalten, Einträge in der Reihenfolge
  E-012 bis E-020 sortiert. Jeder Branch hatte seine Entscheidungen ans Dateiende
  angehängt, während `main` dort inzwischen andere stehen hatte.
- `Jenkinsfile` (#11): der `environment`-Block enthält jetzt beides -- die
  Grenzwerte des Health-Gates aus #10 und `COVERAGE_SCHWELLE`,
  `JUNIT_REPORT_VERSION` und `TEST_DB` aus #11.
- `README.md` (#11): die Stage-Tabelle aus #10 mit der Spalte *Arbeitet in*
  bleibt; die Zeile *Unit-Tests* trägt die Beschreibung aus #11.
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

### Die Deploy-Stages sind noch in keinem Build gelaufen

Die Berechtigung, an der sie zunächst scheiterten, ist inzwischen eingerichtet:

    $ id jenkins
    uid=109(jenkins) gid=112(jenkins) groups=112(jenkins),110(admin),988(docker)

`/home/admin/healthgate` ist für Jenkins damit über die Gruppe beschreibbar,
ohne `sudo`-Rechte. `active-slot.conf`, `historie.tsv` und `vorheriger-slot`
gehören der Gruppe `jenkins` und sind schreibbar; `/home/admin/healthgate/.env`
bleibt mit Rechten 600 unlesbar, die Pipeline benutzt `/etc/healthgate/.env`.

Beim Nachprüfen kamen zwei Dinge heraus, die PR #14 behebt:

1. **git verweigert die Arbeit im fremden Verzeichnis.** `dubious ownership`,
   weil das Verzeichnis `admin` gehört. Die Ausnahme steht jetzt als
   `safe.directory` im `environment`-Block der beiden betroffenen Stages und
   nicht in der gitconfig des Agenten (E-021). Als Umgebungsvariable geprüft:
   `git -C /home/admin/healthgate log` läuft als Benutzer `jenkins` durch.
2. **Der erste Checkout hätte den aktiven Slot zurückgesetzt.** Das Deployment
   steht auf einem Stand, in dem `active-slot.conf` noch versioniert ist; der
   Checkout auf den neuen Stand entfernt die Datei, und aus der Vorlage neu
   angelegt zeigte sie auf `blue`, während Produktion auf `grün` läuft. Die
   Stage sichert die Datei jetzt vorher und stellt sie danach wieder her.
   In einem Klon des Deployment-Verzeichnisses durchgespielt: vorher `green`,
   nach dem Checkout `green`, Arbeitsverzeichnis sauber.

Dazu kam ein dritter Fund, der nichts mit dem Deployment zu tun hat: zwei
Builds auf verschiedenen Branches liefen gleichzeitig gegen denselben
Staging-Stack, und der eine räumte dem anderen die Container weg
(`dependency failed to start: container healthgate-staging-db-1 exited (0)`).
`disableConcurrentBuilds` gilt nur je Job. Staging bekommt jetzt je Branch einen
eigenen Projektnamen, einen vom Docker-Daemon vergebenen Port und ein eigenes
Netz (E-022). In einem Testlauf auf der Maschine geprüft: Port 32770 vergeben,
`/healthz` grün, Produktion unberührt.

Was bleibt: ein echter Durchlauf auf `main`. Erst der zeigt, ob Bespielen,
Umschalten und Beobachtungsfenster zusammen tragen.

**Was stattdessen geprüft wurde**, damit der ungeprüfte Teil so klein wie
möglich bleibt:

| Bestandteil | Nachweis |
|---|---|
| Beide Compose-Dateien | `docker compose config` mit der echten `.env`, fehlerfrei |
| Alle Deploy-Skripte | `bash -n`, Syntax fehlerfrei |
| `wait-healthy.sh container:<name>` | gegen den laufenden Slot grün, gegen einen nicht vorhandenen rot |
| Bootstrap von `active-slot.conf` | in einem Testverzeichnis angelegt und wieder eingelesen |
| Caddy mit Verzeichnis-Mount | eigener Testcontainer: Konfiguration geladen, Weiterleitung auf grün |
| Der Inode-Fall aus E-015 | Datei im Testverzeichnis per `mv` ersetzt, `caddy reload`, Container schaltet auf blau -- der Verzeichnis-Mount sieht die Ersetzung, der Datei-Mount hätte sie nicht gesehen |

Der Testcontainer war nicht am Port 80 und hatte keinen Einfluss auf Produktion;
Produktion lief die ganze Zeit auf grün.

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

## Was als Nächstes zu tun ist

1. **Die vier offenen PRs prüfen und mergen.** Jeder PR gehört der jeweils
   anderen Person zum Review -- sie stammen alle aus derselben Sitzung und haben
   noch niemanden gesehen.
2. **Nach #14 einen Build auf `main` auslösen.** Danach hält die Pipeline bei
   *Freigabe für Produktion* an; die Freigabe gibt eine Person, nicht die
   Pipeline. Die Gruppenzugehörigkeit von `jenkins` ist bereits eingerichtet.
3. **Den ersten vollständigen Durchlauf begleiten.** Beim ersten Deployment nach
   dem Merge von #10 wird der Caddy-Container einmal neu erzeugt, weil sich sein
   Mount ändert. Das dauert Sekunden, ist aber der einzige Moment, in dem der
   Proxy kurz weg ist. Danach läuft der Wechsel wieder ohne Unterbrechung.
4. **Nach dem Merge von #12 das Monitoring neu laden**, damit Grafana das
   Dashboard einliest:

       docker compose -f monitoring/docker-compose.monitoring.yml --env-file .env up -d grafana

5. **Den Rollback vorführen und dabei Last erzeugen.** Ohne Verkehr ist das
   Beobachtungsfenster aussagelos, `observe.sh` sagt das auch:

       deploy/scripts/last-erzeugen.sh http://localhost 5

   Für den Fehlerfall den Chaos-Wert des Zielslots setzen, etwa
   `CHAOS_GREEN=0.3`, und den Build laufen lassen.
