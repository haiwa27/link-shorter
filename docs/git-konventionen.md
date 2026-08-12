# Git-Konventionen und Branching-Modell

Grundlage für Story C-01. Der gelebte Arbeitsablauf ist gleichzeitig der
Nachweis für die Bewertung: die PR-Historie belegt, dass zu zweit gearbeitet
und gegenseitig geprüft wurde.

## Das Modell: trunk-based mit Tags

    main ────●────●────●────●────●────●──────────▶  immer auslieferbar
              \        /      \      /
               ●──●──●         ●──●──●               feature/<story-id>-<titel>
                                     │
                                   v0.2.0            Release = annotierter Tag

- `main` ist geschützt und jederzeit auslieferbar. Jeder Merge kann nach
  Produktion gehen.
- Ein kurzlebiger Branch je Story, angelegt beim Beginn der Arbeit -- nicht
  vorab für alle Stories.
- Lebensdaür höchstens zwei Tage. Wird eine Story größer, wird sie geteilt.
- Ein Release ist ein **annotierter Tag** auf `main`, kein Branch.

### Branch-Arten

| Form | Zweck | Ziel |
|---|---|---|
| `feature/<story-id>-<titel>` | eine Story aus dem Backlog | `main` |
| `fix/<kurztitel>` | Fehler im laufenden Sprint | `main` |
| `hotfix/<kurztitel>` | Produktion ist gestört | `main`, Freigabe darf entfallen |
| `spike/<thema>` | Experiment, etwa PromQL üben | wird verworfen, nie gemerged |
| `chore/release-vX.Y.Z` | Changelog eines Releases | `main`, von `release.sh` erzeugt |

Branchnamen bleiben bei ASCII. Umlaute in Refs führen je nach Werkzeug und
Dateisystem zu Überraschungen. In Commit-Nachrichten, Kommentaren und
Dokumentation werden Umlaute geschrieben.

    tools/story-start.sh R-04 "Beobachtungsfenster nach dem Umschalten"
    tools/story-pr.sh

## Warum kein Git Flow

Git Flow mit `develop` und Release-Branches wurde für Software mit geplanten
Versionen und mehreren parallel gepflegten Ständen entworfen: Installer,
Produkte beim Kunden vor Ort. Dieses Projekt liefert kontinuierlich aus.

Konkret spricht dagegen:

1. **Der Auslöser fehlt.** Mit einem `develop` dazwischen ist `main` nicht mehr
   der Stand, der nach Produktion geht. Die Pipeline hätte keinen natürlichen
   Anlass, das Health-Gate laufen zu lassen -- das Kernmerkmal des Projekts.
2. **Zwei Integrationszweige verdoppeln die Arbeit.** Jeder Fix muss in
   `develop` und in den Release-Branch. Bei zwei Personen und vier Wochen ist
   das nur Aufwand ohne Gegenwert.
3. **Release-Branches lösen ein Problem, das wir nicht haben.** Sie erlauben,
   an Version 2.1 weiterzuarbeiten, während 2.0 noch gepflegt wird. Wir pflegen
   genau einen Stand.
4. **Der Autor selbst rät ab.** Vincent Driessen hat seinem Artikel von 2010
   später einen Hinweis vorangestellt: für kontinuierlich ausgelieferte Software
   ist ein einfacheres Modell die bessere Wahl.

Was Release-Branches liefern sollten -- Nachvollziehbarkeit, welcher Stand wann
ausgeliefert wurde -- liefern hier Tags plus die Deployment-Historie unter
`deploy/state/historie.tsv`. Ohne den zweiten Integrationszweig.

## Commits: Conventional Commits

Form: `typ(bereich): Betreff`

    feat(shortener): Wunsch-Slug beim Anlegen erlauben
    fix(rollback): Rollback nur nach erfolgtem Umschalten auslösen
    docs(architektur): Begründung für Blue/Green ergänzen

| Typ | Bedeutung | Changelog |
|---|---|---|
| `feat` | neue Funktion | Neu |
| `fix` | Fehlerbehebung | Behoben |
| `perf` | Verbesserung ohne neue Funktion | Verbessert |
| `refactor` | Umbau ohne Verhaltensänderung | Umgebaut |
| `docs` | Dokumentation | Dokumentation |
| `ci`, `build` | Pipeline, Build, Abhängigkeiten | Pipeline und Build |
| `test` | Tests | Tests |
| `chore` | Sonstiges | erscheint nicht |

Bereiche: `shortener`, `pipeline`, `deploy`, `monitoring`, `rollback`, `web`,
`docs`.

Regeln: Betreff im Imperativ, höchstens 72 Zeichen, kein Punkt am Ende, zweite
Zeile leer. Der Text sagt **warum**, das **was** steht im Diff. Story-ID nennen.

Der Grund für die Form ist nicht Formalismus: `tools/release.sh` erzeugt daraus
das Changelog. Ohne einheitliche Form gibt es kein Changelog.

## Releases

    tools/release.sh minor

Das Skript prüft, ob `main` sauber und grün ist, berechnet die nächste Version,
stellt das Changelog aus den Commits zusammen, setzt einen annotierten Tag und
legt das Changelog als Pull Request ab.

Für dieses Projekt sinnvoll: ein Release am Ende jedes Sprints. Damit habt ihr
vier Tags, die den Fortschritt belegen, und im Vortrag ein Changelog, das ihr
nicht von Hand geschrieben habt.

## Pull Requests

- Jeder PR wird von der jeweils anderen Person geprüft. Selbstfreigabe gilt
  nicht, auch nicht bei Zeitdruck.
- Die Akzeptanzkriterien der Story stehen im PR und werden abgehakt.
- Ein Nachweis gehört dazu: Ausgabe, Bild oder Build-Nummer.
- Merge erst, wenn die Pipeline grün ist. Squash-Merge, damit die Historie auf
  `main` je Story einen Commit hat -- das hält das Changelog lesbar.

## Hooks

Aktiv über `core.hooksPath=.githooks`, liegen im Repository und gelten für beide.

| Hook | Prüfung |
|---|---|
| `pre-commit` | keine `.env`, keine verbotenen Muster, `gofmt`, `go vet` |
| `commit-msg` | Conventional-Commit-Form, Länge, leere zweite Zeile |
| `pre-push` | kein direkter Push auf `main`, Unit-Tests grün |

Der `pre-push`-Hook wertet die tatsächlich gepushten Refs aus. Ein Tag zu pushen
ist deshalb erlaubt, auch wenn man auf `main` steht.

Die Sperrliste `.git-verbotene-muster` wird **nicht** versioniert und enthält
lokal die echten Domains, Namen und Rufnummern, die in versionierten Dateien
nichts zu suchen haben.

Ein Hook lässt sich mit `--no-verify` umgehen. Das ist gelegentlich nötig -- beim
erstmaligen Push von `main` etwa -- muss aber im PR begründet werden.

## Wer prüft was

Reviews laufen über Kreuz, damit beide den ganzen Aufbau kennen. Wer die
Anwendung schreibt, prüft die Pipeline-Änderungen und umgekehrt. Der Zweck ist
nicht Fehlersuche, sondern dass am Ende keine Person allein einen Teil des
Projekts erklären kann.
