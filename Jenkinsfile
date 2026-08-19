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
				dir('app') {
					sh '''
						go test ./... -v -covermode=count -coverprofile=coverage.out 2>&1 | tee test.log
						go tool cover -func=coverage.out | tail -1
					'''
					// TODO(Q-03): Coverage-Schwelle scharf stellen, sobald die
					// Handler fertig sind. Startwert bewusst niedrig, damit der
					// erste grüne Build nicht am Gate scheitert.
					sh '''
						SCHWELLE=40
						IST=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d '%')
						echo "Coverage: ${IST}% (Mindestwert ${SCHWELLE}%)"
						awk -v i="$IST" -v s="$SCHWELLE" 'BEGIN{exit !(i<s)}' && {
							echo "Coverage unter dem Mindestwert"; exit 1;
						}
						exit 0
					'''
				}
			}
			post {
				always {
					// TODO(Q-06): go-junit-report einbinden, damit Jenkins die
					// Testergebnisse strukturiert anzeigt statt nur als Text.
					archiveArtifacts artifacts: 'app/coverage.out, app/test.log', allowEmptyArchive: true
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
