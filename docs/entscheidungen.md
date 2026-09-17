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

## E-008: pgx statt database/sql für den Zugriff auf PostgreSQL

- **Alternativen:** `database/sql` mit `lib/pq` oder `pgx` im
  `database/sql`-Modus, ORM wie GORM
- **Entscheidung:** `github.com/jackc/pgx/v5` mit `pgxpool` direkt
- **Begründung:** `database/sql` ist eine Abstraktion über mehrere Datenbanken,
  die hier niemand braucht -- dafür kostet sie das Binärprotokoll von
  PostgreSQL und die typisierten Fehler. Genau die werden gebraucht: der
  SQLSTATE 23505 unterscheidet "Slug belegt" von "Datenbank kaputt", und ohne
  ihn müsste die Fehlermeldung als Text geprüft werden. Ein ORM wäre für vier
  Abfragen mehr Lernaufwand als Ersparnis; das Projekt soll SQL zeigen, nicht
  verstecken.

## E-009: Verbindungspool ohne Verbindungsaufbau beim Start

- **Alternativen:** beim Start verbinden und bei Misserfolg abbrechen
- **Entscheidung:** Pool anlegen, nicht verbinden; `Pruefen` macht den Ping
- **Begründung:** Ein Abbruch beim Start hieße in Produktion: Container in der
  Neustartschleife, beide Slots gleichzeitig weg, sobald die Datenbank kurz
  hakt. So bleibt der Prozess stehen, `/healthz` meldet ehrlich 503, Caddy
  nimmt den Slot aus dem Verkehr und er kommt von allein zurück. Die
  Konfiguration bricht weiterhin hart ab, wenn `HEALTHGATE_DB_URL` in `prod`
  fehlt -- ein Tippfehler in der Konfiguration soll auffallen, eine kurz
  abwesende Datenbank nicht eskalieren.

## E-010: Schema über die Migration beim Datenbankstart statt Migrationswerkzeug

- **Alternativen:** golang-migrate als eigene Pipeline-Stage, Migration beim
  Anwendungsstart aus dem Binary heraus
- **Entscheidung:** `app/migrations/0001_init.sql` wird über
  `docker-entrypoint-initdb.d` angewendet; die Datenbanktests wenden dieselbe
  Datei an
- **Begründung:** Für genau eine Tabelle ist ein Migrationswerkzeug Aufwand
  ohne Gegenwert. Wichtig ist, dass es nur eine Schemaquelle gibt: die Tests
  lesen dieselbe SQL-Datei, statt das Schema noch einmal zu definieren. Die
  Grenze ist bekannt und bewusst in Kauf genommen: `initdb` läuft nur bei
  leerem Volume, eine spätere Migration 0002 käme auf einer bestehenden
  Datenbank nicht an. Sobald die zweite Migration ansteht, kommt golang-migrate
  als eigene Stage vor dem Umschalten.

## E-011: Datenbanktests gegen eine echte Wegwerf-Instanz statt gegen Attrappen

- **Alternativen:** Attrappe der Schnittstelle, sqlmock
- **Entscheidung:** PostgreSQL-Container in der Pipeline, Tests über
  `HEALTHGATE_TEST_DB_URL` zuschaltbar
- **Begründung:** Eine Attrappe bestätigt nur, dass der Code die Abfragen
  absetzt, die er absetzt. Sie belegt weder, dass das SQL gültig ist, noch dass
  ein doppelter Slug wirklich als `ErrBelegt` ankommt -- und genau diese
  Übersetzung ist der Teil, der schiefgehen kann. Ohne gesetzte Variable
  überspringen die Tests sich selbst, damit der lokale Lauf ohne Docker
  weiterhin grün ist statt rot.
