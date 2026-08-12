# Backlog

Der vollständige Backlog mit Akzeptanzkriterien, Story Points und
Sprintzuordnung liegt in der Projektplanung.

Kurzübersicht der Epics:

| Epic | Inhalt |
|---|---|
| P | Produkt: Link-Shortener (P-01 bis P-05) |
| Q | Qualitätssicherung: Unit, Coverage, Playwright (Q-01 bis Q-07) |
| C | Build- und Deploy-Pipeline (C-01 bis C-08) |
| O | Observability: Health, Metriken, Dashboards (O-01 bis O-06) |
| R | Progressive Delivery und Rollback (R-01 bis R-08) |
| D | Dokumentation und Abgabe (D-01 bis D-03) |

Offene Stellen im Code sind mit der jeweiligen Story-ID markiert:

    grep -rn 'TODO(' --include='*.go' --include='*.ts' --include='Jenkinsfile' .
