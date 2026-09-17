// Pipeline für healthgate. Liegt bewusst im Repository, damit jede Änderung
// an der Auslieferung genauso reviewt wird wie eine Änderung am Code (C-03).

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
		// TODO(C-05): eigene Registry eintragen. Platzhalter, damit hier kein
		// echter Hostname im Repository steht.
		REGISTRY     = 'registry.beispiel.de/healthgate'
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
		// TODO(R-04): Grenzwerte gemeinsam festlegen und im Vortrag begründen.
		OBSERVE_DAUER     = '120'
		OBSERVE_INTERVALL = '10'
		OBSERVE_SCHWELLE  = '0.05'
		PROMETHEUS_URL    = 'http://localhost:9090'
	}

	stages {

		stage('Vorbereitung') {
			steps {
				sh 'git rev-parse --short HEAD'
				sh 'go version && docker --version && jq --version'
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
				sh 'docker build -f app/Dockerfile -t ${REGISTRY}:${SHA} .'
			}
		}

		stage('Image veröffentlichen') {
			steps {
				// TODO(C-08): Zugangsdaten über withCredentials einbinden.
				// Niemals Token im Jenkinsfile oder im Log.
				echo 'TODO: docker push ${REGISTRY}:${SHA}'
			}
		}

		stage('Staging ausliefern') {
			steps {
				sh '''
					HEALTHGATE_VERSION=${SHA} \
						docker compose -f deploy/docker-compose.staging.yml --env-file .env up -d --build
				'''
				sh 'deploy/scripts/wait-healthy.sh ${STAGING_URL} 3 30'
			}
		}

		stage('E2E-Tests gegen Staging') {
			steps {
				dir('tests/e2e') {
					sh 'npm ci || npm install'
					sh 'npx playwright install --with-deps chromium'
					sh 'BASIS_URL=${STAGING_URL} npx playwright test'
				}
			}
			post {
				always {
					// Playwright schreibt den JUnit-Bericht nur mit gesetztem
					// CI; ohne ihn bleibt die Auswertung leer statt rot.
					junit testResults: 'tests/e2e/test-results/junit.xml', allowEmptyResults: true
					archiveArtifacts artifacts: 'tests/e2e/playwright-report/**, tests/e2e/test-results/**',
						allowEmptyArchive: true
				}
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

		stage('Zielslot bespielen') {
			when { branch 'main' }
			steps {
				script {
					env.ZIEL_SLOT = sh(script: 'deploy/scripts/target-slot.sh', returnStdout: true).trim()
					env.ALT_SLOT  = sh(script: 'deploy/scripts/active-slot.sh', returnStdout: true).trim()
					echo "Aktiv: ${env.ALT_SLOT} -> bespiele: ${env.ZIEL_SLOT}"
				}
				sh '''
					if [ "$ZIEL_SLOT" = "blue" ]; then
						VERSION_BLUE=${SHA} docker compose -f deploy/docker-compose.prod.yml \
							--env-file .env up -d --build app-blue
					else
						VERSION_GREEN=${SHA} docker compose -f deploy/docker-compose.prod.yml \
							--env-file .env up -d --build app-green
					fi
				'''
			}
		}

		stage('Prüfung vor dem Umschalten') {
			when { branch 'main' }
			steps {
				// Story R-03: der neue Slot muss sich mehrfach gesund melden,
				// bevor er überhaupt Verkehr sieht. Hier wird der Container
				// direkt angesprochen, nicht über den Reverse Proxy.
				sh '''
					PORT=$(docker port healthgate-prod-app-${ZIEL_SLOT}-1 8080/tcp | head -1 | cut -d: -f2)
					deploy/scripts/wait-healthy.sh "http://localhost:${PORT}" 3 30
				'''
			}
		}

		stage('Umschalten') {
			when { branch 'main' }
			steps {
				sh 'deploy/scripts/switch-slot.sh ${ZIEL_SLOT}'
				script { env.UMGESCHALTET = 'ja' }
			}
		}

		stage('Beobachtungsfenster') {
			when { branch 'main' }
			steps {
				// Story R-04: das eigentliche Gate. Exitcode 1 trägt in den
				// post-Block und löst dort den Rollback aus.
				sh 'deploy/scripts/observe.sh ${ZIEL_SLOT}'
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
					sh "deploy/scripts/rollback.sh 'Beobachtungsfenster verletzt, Build ${env.BUILD_NUMBER}'"
				} else {
					echo 'Kein Umschalten erfolgt, kein Rollback noetig.'
				}
			}
		}
		success {
			echo "Version ${env.SHA} aktiv auf Slot ${env.ZIEL_SLOT ?: 'staging'}"
		}
		always {
			sh 'deploy/scripts/active-slot.sh || true'
		}
	}
}
