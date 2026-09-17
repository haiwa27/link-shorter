# Entscheidungen

Kurzform: Was wurde entschieden, welche Alternativen gab es, warum so.
Jede Entscheidung, die im Vortrag eine Frage auslösen könnte, gehört hierher.

## E-001: Blue/Green statt fortlaufender Aktualisierung

- **Alternativen:** Container ersetzen (rolling update), Canary mit
  prozentualer Verkehrsaufteilung
- **Entscheidung:** Blue/Green
- **Begründung:** Der Rollback ist eine Umlenkung und kein Neubau. Canary
  wäre fachlich reizvoll, braucht aber gewichtetes Routing und deutlich mehr
  Verkehr, um statistisch etwas auszusagen -- in vier Wochen nicht sinnvoll
  abzusichern.

## E-002: Grenzwert auf Basis der Fehlerrate, nicht der absoluten Fehlerzahl

- **Alternativen:** absolute Zahl von 5xx im Fenster
- **Entscheidung:** Anteil der 5xx an allen Anfragen
- **Begründung:** Eine absolute Zahl hängt vom Verkehrsaufkommen ab und ist
  bei wenig Last blind, bei viel Last übersensibel. Ergänzt wird sie durch
  eine Mindestlast: unter dieser Schwelle meldet `observe.sh`, dass das Fenster
  aussagelos ist, statt Erfolg vorzutäuschen.

## E-003: Chaos-Schalter nur über Umgebungsvariable

- **Alternativen:** Admin-Endpunkt zum Umschalten zur Laufzeit
- **Entscheidung:** Umgebungsvariable, nur beim Deployment setzbar
- **Begründung:** Ein Endpunkt, der Fehler auslöst, ist ein Angriffspunkt.
  Über die Variable wird der Fehlerfall zu einer Eigenschaft des Deployments
  und damit reproduzierbar und nachvollziehbar.

## E-004: Metriken ohne externe Bibliothek

- **Alternativen:** prometheus/client_golang
- **Entscheidung:** eigenes kleines Paket im Textformat
- **Begründung:** Das Projektgerüst baut damit ohne Netzzugriff und der erste
  grüne Build hängt nicht an einem Modul-Download. Sobald echte Histogramme
  gebraucht werden, wird getauscht -- die Schnittstelle bleibt gleich.
  Nebeneffekt: das Exposition-Format wird einmal wirklich verstanden.

## E-005: 302 statt 301 bei der Weiterleitung

- **Alternativen:** 301 (dauerhaft)
- **Entscheidung:** 302 (temporär)
- **Begründung:** Eine dauerhafte Weiterleitung wird vom Browser gecacht.
  Beim Wiederholen der E2E-Tests und beim Zählen der Aufrufe wäre das eine
  stille Fehlerqülle. Die Akzeptanzkriterien von Story P-02 wurden entsprechend
  angepasst.

## E-006: Trunk-based Entwicklung statt Git Flow

- **Alternativen:** Git Flow mit `develop` und Release-Branches, GitLab Flow
  mit Umgebungs-Branches
- **Entscheidung:** kurzlebige Branches direkt auf `main`, Releases als Tags
- **Begründung:** Mit einem Integrationszweig zwischen Feature und Produktion
  ist `main` nicht mehr der ausgelieferte Stand, und das Health-Gate verliert
  seinen Auslöser. Git Flow adressiert mehrere parallel gepflegte Versionen --
  ein Problem, das dieses Projekt nicht hat. Nachvollziehbarkeit liefern Tags
  und die Deployment-Historie. Ausführlich in docs/git-konventionen.md.

## E-007: Conventional Commits

- **Alternativen:** freie Commit-Nachrichten mit Längenbegrenzung
- **Entscheidung:** `typ(bereich): Betreff`, erzwungen im commit-msg-Hook
- **Begründung:** Das Changelog entsteht dadurch automatisch aus der Historie
  statt von Hand am Ende des Sprints. Der Zwang ist der Preis dafür; ohne
  einheitliche Form gäbe es kein maschinell erzeugtes Changelog.

## E-019: Dashboard als Provisioning-Datei statt in der Oberfläche geklickt

- **Alternativen:** Dashboard in Grafana anlegen und exportieren, wenn jemand
  danach fragt
- **Entscheidung:** `monitoring/grafana/provisioning/dashboards/healthgate.json`
  im Repository, Grafana lädt es beim Start
- **Begründung:** Ein geklicktes Dashboard lebt im Grafana-Volume. Es ist damit
  nicht reviewbar, nicht reproduzierbar und beim nächsten frischen Aufsetzen
  weg. Als Datei im Repository ist es dieselbe Art von Artefakt wie die
  Alarmregeln. Die Datenquelle bekommt dafür eine feste `uid`: ohne sie vergibt
  Grafana bei jeder Neuinstallation eine andere, und das Dashboard zeigte auf
  eine Datenquelle, die es nicht gibt.

## E-020: Deployment-Marker aus den Metriken statt über die Grafana-API

- **Alternativen:** Die Pipeline schreibt nach dem Umschalten eine Annotation
  über die HTTP-API von Grafana
- **Entscheidung:** zwei Annotationen im Dashboard, beide als PromQL-Abfrage --
  `resets(healthgate_uptime_seconds[2m])` markiert den Neustart eines Slots,
  ein `unless`-Ausdruck über der Anfragerate markiert den Moment, in dem ein
  Slot Verkehr bekommt
- **Begründung:** Der API-Weg bräuchte ein Grafana-Token in den
  Jenkins-Credentials. Das ist Konfiguration in der Oberfläche und damit genau
  der nicht versionierte Zustand, den dieses Projekt vermeidet; ausserdem wäre
  der Marker gesetzt, auch wenn das Deployment danach zurückgerollt wird. Aus
  den Metriken abgeleitet zeigt der Marker, was tatsächlich passiert ist, und
  nicht, was die Pipeline gemeldet hat. Die Health-Checks helfen dabei: Caddy
  prüft nur den Slot, der Verkehr bekommt, deshalb ist das Umschalten auch ohne
  Nutzerlast sichtbar.
