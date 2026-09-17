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
