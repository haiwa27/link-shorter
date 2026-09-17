// Pipeline für healthgate. Liegt bewusst im Repository, damit jede Änderung
// an der Auslieferung genauso reviewt wird wie eine Änderung am Code (C-03).
//
// Die tragende Regel dieser Datei: gebaut und getestet wird im Jenkins-Workspace,
// ausgeliefert wird ausschliesslich gegen DEPLOY_DIR. Der Workspace ist bei
// jedem Lauf ein frischer Checkout und kennt den laufenden Zustand nicht --
// welcher Slot gerade Verkehr bekommt, steht nur im Deployment-Verzeichnis, aus
// dem Caddy seine Konfiguration mountet (Entscheidung E-012).

pipeline {
	agent any

	options {
		timestamps()
		buildDiscarder(logRotator(numToKeepStr: '30'))
		timeout(time: 45, unit: 'MINUTES')
		disableConcurrentBuilds()
	}

	environment {
		SHA          = "${env.GIT_COMMIT?.take(7) ?: 'dev'}"
		// Das Image trägt den Commit im Namen und wird genau einmal gebaut.
		// Der Name ist zugleich die Variable, die beide Compose-Dateien lesen;
		// so kann keine Stage versehentlich ein anderes Artefakt ausliefern.
		HEALTHGATE_IMAGE = "healthgate:${env.GIT_COMMIT?.take(7) ?: 'dev'}"
		// Version der E2E-Werkzeuge. Playwright läuft im eigenen Container,
		// weil die Maschine Node 18 mitbringt und Playwright 20 verlangt.
		PLAYWRIGHT_IMAGE = 'mcr.microsoft.com/playwright:v1.62.1-noble'
		// TODO(C-05): eigene Registry eintragen. Platzhalter, damit hier kein
		// echter Hostname im Repository steht.
		REGISTRY     = 'registry.beispiel.de/healthgate'

		// Das laufende Deployment. Jede Stage, die den aktiven Slot liest oder
		// ändert, arbeitet hier und nicht im Workspace.
		DEPLOY_DIR   = '/home/admin/healthgate'
		SKRIPTE      = '/home/admin/healthgate/deploy/scripts'
		// Zugangsdaten liegen ausserhalb des Workspace: der wird bei jedem Lauf
		// neu ausgecheckt und darf keine Geheimnisse enthalten (E-014).
		ENV_DATEI    = '/etc/healthgate/.env'

		STAGING_URL  = 'http://localhost:8081'
		PROD_URL     = 'http://localhost'

		// Mindest-Coverage. Der Wert liegt bewusst unter dem aktuellen Stand:
		// er soll einen Einbruch melden, nicht jede dritte Nachkommastelle
		// (Story Q-03, Entscheidung E-017).
		COVERAGE_SCHWELLE = '65'
		// Version fest gepinnt: ein Werkzeug, das den Build zum Scheitern
		// bringen kann, darf sich nicht unter der Hand ändern.
		JUNIT_REPORT_VERSION = 'v2.1.0'
		// Wegwerf-Datenbank für die Tests des PostgreSQL-Speichers. Der Name
		// enthält den Build, damit parallele Jobs sich nicht überschreiben.
		TEST_DB = "healthgate-test-db-${env.BUILD_TAG?.replaceAll('[^A-Za-z0-9_.-]', '-') ?: 'lokal'}"

		// Grenzwerte des Health-Gates. Sie stehen hier und nicht in der .env,
		// weil sie zur Pipeline gehören und versioniert sein müssen (Story R-04).
		OBSERVE_DAUER        = '120'
		OBSERVE_INTERVALL    = '10'
		OBSERVE_SCHWELLE     = '0.05'
		OBSERVE_MIN_ANFRAGEN = '5'
		PROMETHEUS_URL       = 'http://localhost:9090'
	}

	stages {

		stage('Vorbereitung') {
			steps {
				sh 'git rev-parse --short HEAD'
				sh 'go version && docker --version && jq --version'
				sh '''
					test -r "${ENV_DATEI}" || {
						echo "FEHLER: ${ENV_DATEI} ist fuer den Jenkins-Benutzer nicht lesbar."
						echo "        Die Datei gehoert der Gruppe jenkins mit Rechten 640."
						exit 1
					}
					echo "Umgebungsdatei: ${ENV_DATEI}"
				'''
				script {
					currentBuild.displayName = "#${env.BUILD_NUMBER} ${env.SHA}"
				}
			}
		}

		stage('Statische Analyse') {
			steps {
				dir('app') {
					sh 'go vet ./...'
					// gofmt meldet nur, es formatiert nicht. Ein unformatiertes
					// File soll auffallen, nicht unbemerkt korrigiert werden.
					sh 'test -z "$(gofmt -l .)" || { gofmt -l .; echo "Nicht formatiert"; exit 1; }'
				}
			}
		}

		stage('Unit-Tests') {
			steps {
				sh '''#!/usr/bin/env bash
					set -euo pipefail

					export PATH="$PATH:$(go env GOPATH)/bin"
					command -v go-junit-report >/dev/null 2>&1 ||
						go install "github.com/jstemmer/go-junit-report/v2@${JUNIT_REPORT_VERSION}"

					# Ohne echte Datenbank überspringen sich die Tests des
					# PostgreSQL-Speichers selbst. Die Coverage-Zahl wiese dann
					# eine Prüfung aus, die gar nicht stattgefunden hat.
					docker rm -f "$TEST_DB" >/dev/null 2>&1 || true
					docker run -d --name "$TEST_DB" -P \
						-e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=test \
						postgres:16-alpine >/dev/null
					for versuch in $(seq 1 30); do
						docker exec "$TEST_DB" pg_isready -U test -d test >/dev/null 2>&1 && break
						sleep 1
					done
					PORT="$(docker port "$TEST_DB" 5432/tcp | head -1 | sed 's/.*://')"
					export HEALTHGATE_TEST_DB_URL="postgres://test:test@127.0.0.1:${PORT}/test?sslmode=disable"

					cd app
					# Der Exitcode wird festgehalten statt sofort ausgewertet:
					# der Bericht soll auch dann entstehen, wenn Tests
					# fehlschlagen -- sonst zeigt Jenkins beim roten Build gerade
					# die Ergebnisse nicht an, die man sehen will.
					set +e
					go test ./... -v -covermode=count -coverprofile=coverage.out 2>&1 | tee test.log
					STATUS=${PIPESTATUS[0]}
					set -e

					go-junit-report -set-exit-code < test.log > test-report.xml || true
					go tool cover -func=coverage.out | tail -1

					if [ "$STATUS" -ne 0 ]; then
						echo "Unit-Tests fehlgeschlagen"
						exit 1
					fi
				'''
				dir('app') {
					sh '''#!/usr/bin/env bash
						set -euo pipefail
						IST=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d '%')
						echo "Coverage: ${IST}% (Mindestwert ${COVERAGE_SCHWELLE}%)"
						if awk -v i="$IST" -v s="$COVERAGE_SCHWELLE" 'BEGIN{exit !(i<s)}'; then
							echo "Coverage unter dem Mindestwert"
							exit 1
						fi
					'''
				}
			}
			post {
				always {
					// Story Q-06: Jenkins zeigt die Testergebnisse damit als
					// Tabelle mit Verlauf statt als Textwand im Konsolenprotokoll.
					junit testResults: 'app/test-report.xml', allowEmptyResults: false
					archiveArtifacts artifacts: 'app/coverage.out, app/test.log, app/test-report.xml',
						allowEmptyArchive: true
					sh 'docker rm -f "$TEST_DB" >/dev/null 2>&1 || true'
				}
			}
		}

		stage('Image bauen') {
			steps {
				// Zwei Namen für dasselbe Image: der lokale für die Auslieferung
				// auf dieser Maschine, der Registry-Name für den späteren Push.
				sh 'docker build -f app/Dockerfile -t ${HEALTHGATE_IMAGE} -t ${REGISTRY}:${SHA} .'
			}
		}

		stage('Image veröffentlichen') {
			steps {
				// TODO(C-05): Zugangsdaten über withCredentials einbinden.
				// Niemals Token im Jenkinsfile oder im Log.
				echo "TODO: docker push ${REGISTRY}:${SHA}"
			}
		}

		stage('Staging ausliefern') {
			steps {
				// Staging läuft im Workspace: es ist eine Testumgebung und hält
				// keinen Zustand, den ein frischer Checkout verlieren könnte.
				sh '''
					HEALTHGATE_VERSION=${SHA} docker compose \
						-f deploy/docker-compose.staging.yml \
						--env-file "${ENV_DATEI}" up -d
				'''
				sh 'deploy/scripts/wait-healthy.sh ${STAGING_URL} 3 30'
			}
		}

		stage('E2E-Tests gegen Staging') {
			steps {
				// Playwright läuft im mitgelieferten Container: die Browser sind
				// darin bereits installiert und passen zur Version aus dem
				// Lockfile. Die Maschine selbst bleibt unangetastet (E-016).
				// --network host, damit Staging unter localhost:8081 erreichbar
				// ist; der Lauf als aufrufender Benutzer, damit die Artefakte
				// nicht root gehören und Jenkins sie archivieren kann.
				sh '''
					docker run --rm --network host \
						-v "$PWD/tests/e2e":/e2e -w /e2e \
						-u "$(id -u):$(id -g)" -e HOME=/tmp \
						-e CI=true -e BASIS_URL=${STAGING_URL} \
						${PLAYWRIGHT_IMAGE} \
						sh -c "npm ci && npx playwright test"
				'''
			}
			post {
				always {
					// Playwright schreibt den JUnit-Bericht nur mit gesetztem
					// CI; ohne ihn bleibt die Auswertung leer statt rot.
					junit testResults: 'tests/e2e/test-results/junit.xml', allowEmptyResults: true
					archiveArtifacts artifacts: 'tests/e2e/playwright-report/**, tests/e2e/test-results/**',
						allowEmptyArchive: true
					sh 'docker compose -f deploy/docker-compose.staging.yml down || true'
				}
			}
		}

		stage('Deployment-Verzeichnis prüfen') {
			when { branch 'main' }
			steps {
				// Lieber hier scheitern als nach der Freigabe mitten im
				// Umschalten. Die Meldung nennt beim Fehlschlag den Grund.
				sh '''
					test -w "${DEPLOY_DIR}" || {
						echo "FEHLER: ${DEPLOY_DIR} ist fuer den Jenkins-Benutzer nicht beschreibbar."
						echo "        Einmalig auf der Maschine: sudo usermod -aG admin jenkins"
						echo "        und anschliessend sudo systemctl restart jenkins."
						exit 1
					}
					echo "Deployment-Verzeichnis: ${DEPLOY_DIR}"
					git -C "${DEPLOY_DIR}" --no-pager log --oneline -1
				'''
			}
		}

		stage('Freigabe für Produktion') {
			when { branch 'main' }
			steps {
				// Story C-07: bewusste menschliche Entscheidung vor Produktion.
				timeout(time: 15, unit: 'MINUTES') {
					input message: "Version ${env.SHA} nach Produktion ausliefern?", ok: 'Ausliefern'
				}
			}
		}

		stage('Deployment-Verzeichnis aktualisieren') {
			when { branch 'main' }
			steps {
				// Ohne diesen Schritt liefe ein neues Image gegen die
				// Compose-Datei und die Skripte eines alten Stands. Der aktive
				// Slot bleibt dabei unberührt: active-slot.conf ist nicht
				// versioniert und wird von einem Checkout nicht angefasst.
				sh '''
					cd "${DEPLOY_DIR}"
					git fetch --no-tags origin main
					git checkout -f "${GIT_COMMIT}"
					test -f deploy/caddy/active-slot.conf ||
						cp deploy/caddy/active-slot.conf.vorlage deploy/caddy/active-slot.conf
					git --no-pager log --oneline -1
					echo "Aktiver Slot bleibt: $(deploy/scripts/active-slot.sh)"
				'''
			}
		}

		stage('Reverse Proxy abgleichen') {
			when { branch 'main' }
			steps {
				// --no-deps ist hier wesentlich: ohne das Flag zöge Caddy seine
				// depends_on mit und Compose stellte beide Slots gleichzeitig
				// auf das neue Image um. Damit wäre Blue/Green aufgehoben und
				// es gäbe kein Ziel mehr für einen Rollback.
				sh '''
					cd "${DEPLOY_DIR}"
					docker compose -f deploy/docker-compose.prod.yml \
						--env-file "${ENV_DATEI}" up -d --no-deps caddy
				'''
			}
		}

		stage('Zielslot bespielen') {
			when { branch 'main' }
			steps {
				script {
					env.ZIEL_SLOT = sh(script: '${SKRIPTE}/target-slot.sh', returnStdout: true).trim()
					env.ALT_SLOT  = sh(script: '${SKRIPTE}/active-slot.sh', returnStdout: true).trim()
					echo "Aktiv: ${env.ALT_SLOT} -> bespiele: ${env.ZIEL_SLOT}"
				}
				sh '''
					cd "${DEPLOY_DIR}"
					if [ "$ZIEL_SLOT" = "blue" ]; then
						VERSION_BLUE=${SHA} docker compose \
							-f deploy/docker-compose.prod.yml --env-file "${ENV_DATEI}" \
							up -d app-blue
					else
						VERSION_GREEN=${SHA} docker compose \
							-f deploy/docker-compose.prod.yml --env-file "${ENV_DATEI}" \
							up -d app-green
					fi
				'''
			}
		}

		stage('Prüfung vor dem Umschalten') {
			when { branch 'main' }
			steps {
				// Story R-03: der neue Slot muss sich mehrfach gesund melden,
				// bevor er überhaupt Verkehr sieht. Geprüft wird der Container
				// selbst, nicht der Reverse Proxy -- über den Proxy antwortete
				// noch der alte Slot und die Prüfung wäre wertlos.
				sh '${SKRIPTE}/wait-healthy.sh container:healthgate-prod-app-${ZIEL_SLOT}-1 3 30'
			}
		}

		stage('Umschalten') {
			when { branch 'main' }
			steps {
				sh '${SKRIPTE}/switch-slot.sh ${ZIEL_SLOT}'
				script { env.UMGESCHALTET = 'ja' }
			}
		}

		stage('Beobachtungsfenster') {
			when { branch 'main' }
			steps {
				// Story R-04: das eigentliche Gate. Exitcode 1 trägt in den
				// post-Block und löst dort den Rollback aus.
				sh '${SKRIPTE}/observe.sh ${ZIEL_SLOT}'
			}
		}
	}

	post {
		failure {
			script {
				if (env.UMGESCHALTET == 'ja') {
					// Story R-05: automatischer Rückschwenk. Der Build bleibt
					// rot -- ein zurückgerolltes Deployment ist kein Erfolg,
					// sondern ein verhinderter Schaden.
					echo "Rollback wird ausgeloest (Slot ${env.ZIEL_SLOT} auffaellig)"
					sh "${env.SKRIPTE}/rollback.sh 'Beobachtungsfenster verletzt, Build ${env.BUILD_NUMBER}'"
				} else {
					echo 'Kein Umschalten erfolgt, kein Rollback noetig.'
				}
			}
		}
		success {
			echo "Version ${env.SHA} aktiv auf Slot ${env.ZIEL_SLOT ?: 'staging'}"
		}
		always {
			// Auf Branches ohne Deploy-Stages ist das Verzeichnis nicht
			// zwingend erreichbar, deshalb ohne Folgen für das Ergebnis.
			sh '${SKRIPTE}/active-slot.sh || true'
		}
	}
}
