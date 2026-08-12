# healthgate

Continuous-Delivery-Strecke mit Blue/Green-Deployment und automatischem,
metrikbasiertem Rollback. Schulprojekt, 4 Sprints.

Der Gegenstand des Projekts ist die Pipeline, nicht die Anwendung. Die
mitgelieferte Anwendung (ein Link-Shortener) ist absichtlich klein und dient
als Objekt, das ausgeliefert, überwacht und im Fehlerfall zurückgerollt wird.

## Kernidee

Nach dem Umschalten des Verkehrs auf die neue Version beobachtet die Pipeline
die Anwendung für ein definiertes Zeitfenster anhand echter Metriken aus
Prometheus. Überschreitet die Fehlerrate den Grenzwert, schwenkt die Pipeline
ohne menschliches Eingreifen auf die vorherige Version zurück.

## Aufbau

    app/            Go-Backend (API, Health, Metriken)
    web/            Frontend (React, Vite, TypeScript)
    deploy/         Compose-Dateien, Caddy, Umschalt- und Rollback-Skripte
    monitoring/     Prometheus, Grafana
    tests/e2e/      Playwright-Suite
    docs/           Architektur, Entscheidungen, Übergabe
    Jenkinsfile     Pipeline, versioniert im Repository

## Schnellstart (lokal)

Nach dem Klonen einmal:

    tools/setup.sh


    cp .env.example .env        # Werte anpassen
    make dev                    # Backend auf :8080
    make test                   # Unit-Tests mit Coverage
    curl localhost:8080/healthz
    curl localhost:8080/metrics

Frontend separat:

    cd web && npm install && npm run dev

## Umgebungen

    make staging-up             # Staging-Stack
    make prod-up                # Produktion mit beiden Slots (blau und grün)
    make monitoring-up          # Prometheus und Grafana

## Umschalten und Rollback

    deploy/scripts/active-slot.sh          # welcher Slot bekommt Verkehr
    deploy/scripts/switch-slot.sh green    # umschalten
    deploy/scripts/observe.sh green        # Beobachtungsfenster, Exitcode 1 bei Verstoss
    deploy/scripts/rollback.sh             # zurück auf den vorherigen Slot

## Nächste Schritte (Sprint 1)

1. `internal/store`: In-Memory-Speicher durch PostgreSQL ersetzen (Story P-01)
2. `internal/handler`: Anlegen, Liste, Löschen fertigstellen (P-01 bis P-05)
3. Jenkins einrichten, Multibranch-Pipeline auf dieses Repository zeigen (C-02)
4. Webhook einrichten, damit jeder Push baut (C-04)
5. Erster automatischer Staging-Deploy (C-06)

Alle offenen Stellen im Code sind mit `TODO(<Story-ID>)` markiert und lassen
sich so finden:

    grep -rn 'TODO(' --include='*.go' --include='*.ts' .

## Konventionen

- Kein Commit direkt auf `main`. Jede Änderung läuft über einen Pull Request
  mit Review durch die jeweils andere Person.
- Keine echten Hostnamen, Zugangsdaten oder Namen in versionierten Dateien,
  auch nicht in Beispieldateien. Platzhalter: `beispiel.de`, `Muster GmbH`.
- Umlaute werden als ä, ö, ü und ß geschrieben: in Kommentaren, Commit-
  Nachrichten, Dokumentation und sichtbaren Texten. Bezeichner im Code und
  Branchnamen bleiben ASCII.
