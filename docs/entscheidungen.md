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

## E-012: Bauen im Workspace, Ausliefern gegen das Deployment-Verzeichnis

- **Alternativen:** alles im Jenkins-Workspace; das Deployment-Verzeichnis per
  Symlink auf den Workspace zeigen lassen
- **Entscheidung:** Build und Tests im Workspace, jede Deploy-Stage arbeitet
  gegen `/home/admin/healthgate`
- **Begründung:** Der Workspace ist bei jedem Lauf ein frischer Checkout und
  weiss nicht, welcher Slot gerade Verkehr bekommt. Caddy mountet
  `active-slot.conf` aus dem Deployment-Verzeichnis. Liefe `switch-slot.sh` im
  Workspace, beschriebe es die Workspace-Kopie: das Umschalten bliebe wirkungslos,
  `observe.sh` beobachtete den falschen Slot und der Build wäre grün, obwohl
  nichts passiert ist. Genau dieser Fehler meldet sich nirgends -- er sieht aus
  wie ein erfolgreiches Deployment. Deshalb steht die Regel im Kopf des
  Jenkinsfile und nicht nur hier.

## E-013: Ein Image bauen und dasselbe ausliefern, statt in Produktion neu zu bauen

- **Alternativen:** `docker compose up --build` in Staging und Produktion, wie
  im Projektgerüst angelegt
- **Entscheidung:** ein Image je Commit, getaggt mit dem Git-SHA; Staging und
  beide Produktionsslots ziehen genau dieses Image über `HEALTHGATE_IMAGE`
- **Begründung:** Baut Produktion neu, ist das ausgelieferte Artefakt nicht das
  getestete. Zwischen beiden Läufen können sich Basis-Images und npm-Pakete
  ändern; die E2E-Tests bezögen sich dann auf etwas, das so nie in Produktion
  läuft. Nebeneffekt: das Umschalten wird schneller, weil im Deployment kein
  Build mehr steckt. Solange es keine Registry gibt (C-05), reicht der lokale
  Docker-Daemon, weil Jenkins und Produktion auf derselben Maschine liegen.

## E-014: .env ausserhalb des Workspace unter /etc/healthgate

- **Alternativen:** .env im Deployment-Verzeichnis, Jenkins-Credentials als
  Secret-File
- **Entscheidung:** `/etc/healthgate/.env`, Eigentümer root, Gruppe jenkins,
  Rechte 640
- **Begründung:** Der Workspace wird bei jedem Lauf neu ausgecheckt und
  gelegentlich aufgeräumt; eine Datei mit Zugangsdaten hat dort nichts zu
  suchen. Der Pfad ist ausserdem unabhängig davon, aus welchem Verzeichnis eine
  Stage läuft. Ein Jenkins-Secret-File wäre die sauberere Lösung, ist aber
  Konfiguration in der Oberfläche und damit nicht versioniert -- die Pipeline
  soll aus dem Repository heraus nachvollziehbar bleiben. Die Werte werden
  ausschliesslich über `--env-file` an Compose gereicht und nie in die Shell
  geladen, damit sie nicht im Build-Log landen.

## E-015: Der aktive Slot ist Laufzeitzustand und wird nicht versioniert

- **Alternativen:** `active-slot.conf` im Repository lassen und beim Checkout
  im Deployment-Verzeichnis von Hand sichern
- **Entscheidung:** Datei in `.gitignore`, im Repository liegt nur
  `active-slot.conf.vorlage`; Caddy mountet das Verzeichnis `deploy/caddy`
  statt der einzelnen Dateien
- **Begründung:** Welcher Slot Verkehr bekommt, ist Zustand wie
  `deploy/state/vorheriger-slot` und gehört in dieselbe Kategorie. Versioniert
  setzte jeder Checkout im Deployment-Verzeichnis den aktiven Slot still auf den
  eingecheckten Wert zurück, und `active-slot.sh` meldete anschliessend einen
  anderen Slot, als tatsächlich Verkehr bekommt. Der Verzeichnis-Mount ist die
  zweite Hälfte desselben Problems: ein Mount auf eine Einzeldatei hängt an
  deren Inode. Ersetzt jemand die Datei, statt sie zu überschreiben -- was jedes
  `git checkout` tut -- sieht der Container weiter den alten Inhalt, und ein
  `caddy reload` lädt fröhlich die alte Konfiguration nach.

## E-016: Playwright im mitgelieferten Container statt auf der Maschine

- **Alternativen:** Node 20 auf der Maschine nachinstallieren,
  `npx playwright install --with-deps` in der Pipeline
- **Entscheidung:** `mcr.microsoft.com/playwright:<version>` als Container, mit
  `--network host` gegen Staging
- **Begründung:** Die Maschine bringt Node 18 mit, Playwright verlangt ab 1.50
  mindestens Node 20. Eine Installation auf der Maschine wäre Zustand, den
  niemand versioniert und der bei der nächsten Neuinstallation fehlt.
  `--with-deps` bräuchte ausserdem root für apt. Der Container bringt Browser
  und Systempakete in der Version mit, die zum Lockfile passt; die Pipeline
  bleibt damit unabhängig davon, was auf dem Agenten installiert ist.

## E-017: Coverage-Schwelle bei 65 Prozent statt bei 80 oder 40

- **Alternativen:** bei 40 Prozent belassen, auf 80 Prozent anheben, Schwelle je
  Paket statt insgesamt
- **Entscheidung:** 65 Prozent über alle Pakete, gemessen mit laufender
  Testdatenbank
- **Begründung:** 40 Prozent lag unter dem tatsächlichen Stand und hätte einen
  Einbruch nie gemeldet -- eine Schwelle, die immer grün ist, prüft nichts.
  80 Prozent erzwänge Tests für `cmd/server`, wo die Verdrahtung von
  Konfiguration und Server steht: solche Tests prüfen, dass man die Zeilen
  nochmal geschrieben hat, und nicht, dass die Anwendung funktioniert. Der
  gemessene Stand liegt bei rund 72 Prozent; 65 lässt Raum für eine neue
  Funktion, die kurz unter ihren Tests herläuft, und meldet trotzdem, wenn ein
  Paket ohne Tests dazukommt. Ein Wert je Paket wäre genauer, verleitet aber
  dazu, die Schwelle dort zu senken, wo Tests fehlen.

## E-018: Testdatenbank in der Pipeline statt übersprungener Datenbanktests

- **Alternativen:** die Datenbanktests in der Pipeline überspringen lassen,
  Testcontainers-Bibliothek, dauerhaft laufende Testdatenbank
- **Entscheidung:** die Stage startet vor den Tests einen
  PostgreSQL-Container und räumt ihn im `post`-Block wieder ab
- **Begründung:** Übersprungene Tests sind schlimmer als fehlende: die
  Coverage-Zahl weist eine Prüfung aus, die nicht stattgefunden hat. Eine
  dauerhaft laufende Testdatenbank wäre Zustand, den niemand pflegt und der
  zwischen Läufen Daten behält. Testcontainers wäre die saubere Lösung, bringt
  aber eine Abhängigkeit für etwas mit, das hier vier Zeilen `docker run` sind.
  Der Container trägt den Build im Namen, damit gleichzeitige Jobs sich nicht
  gegenseitig abräumen.

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

## E-021: safe.directory als Variable der Stage statt in der gitconfig des Agenten

- **Alternativen:** `git config --global --add safe.directory ...` einmalig als
  Jenkins-Benutzer ausführen; das Deployment-Verzeichnis dem Jenkins-Benutzer
  übereignen
- **Entscheidung:** `GIT_CONFIG_COUNT` und `GIT_CONFIG_KEY_0` im
  `environment`-Block der beiden Stages, die im Deployment-Verzeichnis mit git
  arbeiten
- **Begründung:** git verweigert seit 2.35 jede Operation in einem Verzeichnis,
  das jemand anderem gehört. Der Eintrag in der globalen gitconfig läge in
  `/var/lib/jenkins` -- unversioniert, unsichtbar im Review, und nach einer
  Neuinstallation des Agenten wieder weg. Als Variable steht die Ausnahme im
  Jenkinsfile, gilt genau in den zwei Stages, die sie brauchen, und hinterlässt
  auf der Maschine nichts. Das Verzeichnis zu übereignen scheidet aus: es gehört
  dem Menschen, der die Maschine betreibt, und nicht der Pipeline.

## E-022: Eigener Staging-Stack je Branch statt eines gemeinsamen

- **Alternativen:** Builds über das Lockable-Resources-Plugin serialisieren,
  einen gemeinsamen Stack behalten und auf gleichzeitige Builds verzichten
- **Entscheidung:** Compose-Projektname je Branch, Host-Port vom Docker-Daemon
  vergeben, Netzname von Compose abgeleitet
- **Begründung:** `disableConcurrentBuilds` gilt nur je Job. Zwei Branches bauen
  sehr wohl gleichzeitig, und dann räumt der eine Build dem anderen die
  Container weg -- beobachtet als `dependency failed to start: container
  healthgate-staging-db-1 exited (0)`, während ein zweiter Build gerade `down`
  lief. Ein Lock wäre der direktere Weg, braucht aber ein Plugin, das nicht
  installiert ist, und serialisiert Builds, die sich gar nicht stören müssten.
  Der Preis ist ein Datenvolumen je Branch; es bleibt klein und wird beim
  Aufräumen des Branches mit entfernt.

## E-023: Das Deployment-Verzeichnis holt den Stand aus dem Workspace, nicht von GitHub

- **Alternativen:** `git fetch origin main` im Deployment-Verzeichnis, mit den
  Zugangsdaten aus den Jenkins-Credentials über `withCredentials`
- **Entscheidung:** `git fetch "${WORKSPACE}" HEAD`, anschliessend Checkout auf
  den gebauten Commit
- **Begründung:** Der Umweg über GitHub beantwortet die falsche Frage. Was
  ausgeliefert werden soll, ist nicht "der aktuelle Stand von `main`", sondern
  "der Stand, der gerade gebaut und gegen Staging geprüft wurde" -- und der liegt
  bereits im Workspace. Zwischen dem Bauen und dem Umschalten kann `main`
  weiterlaufen; der Fetch von GitHub lieferte dann einen anderen Stand als das
  Image, das dieselbe Stage ausrollt. Nebeneffekte: die Auslieferung braucht
  weder Netz noch Zugangsdaten für das private Repository, und kein Token muss
  durch eine Shell wandern, wo es im Protokoll landen könnte. Aufgefallen ist
  das Ganze durch `fatal: could not read Username for 'https://github.com'` --
  der Jenkins-Benutzer hat schlicht keine Anmeldung, der Git-Plugin-Checkout
  bringt seine eigene mit.

## E-024: Die Slot-Datei gehört beiden Schreibern, nicht dem zuletzt Schreibenden

- **Alternativen:** das Umschalten ausschliesslich der Pipeline überlassen;
  Jenkins per `sudo` als Betriebsbenutzer schreiben lassen
- **Entscheidung:** `umask 002` in `switch-slot.sh`, `chmod 664` beim Anlegen,
  und das setgid-Bit auf `deploy/caddy` und `deploy/state` auf der Maschine
- **Begründung:** `active-slot.conf` wird von zwei Benutzern geschrieben: von
  der Pipeline und von einem Menschen, der von Hand umschaltet oder zurückrollt.
  Legt einer von beiden die Datei neu an, erbt sie dessen Rechte und Gruppe --
  beim ersten Deployment war das `jenkins:jenkins` mit `600`, und danach
  scheiterte jeder Handgriff an `Permission denied`. Besonders unangenehm, weil
  Caddy die Datei als root im Container weiterlesen kann: von aussen sah alles
  gesund aus, nur der Rollback von Hand war still kaputt. Das setgid-Bit sorgt
  dafür, dass neue Dateien die Gruppe des Verzeichnisses erben, `umask 002` und
  `chmod 664` dafür, dass die Gruppe schreiben darf. Der Weg über `sudo` wurde
  verworfen: die Pipeline soll keine erhöhten Rechte bekommen, um eine
  Textdatei zu schreiben.
