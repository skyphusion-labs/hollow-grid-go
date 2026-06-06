// CI/CD for hollow-grid-go (a Hollow Grid world server in Go), on the mindcrime
// Jenkins.
//
// On every push: gofmt + go vet, unit tests, build the Docker image, and run the
// upstream smoke conformance suite against the freshly-built container
// (informational while the port is in progress, so a partial pass never reds the
// build). On main, (re)deploy the container on this host.
//
// Agent (mindcrime, user `jenkins`): go on PATH (/usr/local/bin/go -> goinstall),
// docker usable directly, node for the smoke suite. The smoke suite is the local
// the-hollow-grid checkout's smoke.mjs; the stage SKIPs cleanly if absent.
pipeline {
  agent any

  options {
    timestamps()
    timeout(time: 15, unit: 'MINUTES')
    disableConcurrentBuilds()
  }

  environment {
    IMAGE = 'hollow-grid-go'
    SMOKE = '/home/conrad/dev/the-hollow-grid/smoke.mjs'
    PATH  = "/usr/local/bin:/home/conrad/goinstall/go/bin:${env.PATH}"
  }

  stages {
    stage('Lint & Vet') {
      steps {
        sh '''
          set -e
          go version
          fmt=$(gofmt -l ./cmd ./internal)
          if [ -n "$fmt" ]; then echo "gofmt needed on:"; echo "$fmt"; exit 1; fi
          go vet ./...
        '''
      }
    }

    stage('Test') {
      steps { sh 'go test ./...' }
    }

    stage('Docker Build') {
      steps { sh 'docker build -t "$IMAGE:$GIT_COMMIT" -t "$IMAGE:latest" .' }
    }

    stage('Smoke (conformance, informational)') {
      steps {
        sh '''
          set -e
          if [ ! -f "$SMOKE" ]; then echo "SKIP: smoke suite not found at $SMOKE"; exit 0; fi
          if ! command -v node >/dev/null 2>&1; then echo "SKIP: node not on PATH"; exit 0; fi
          docker rm -f hgg-ci >/dev/null 2>&1 || true
          docker run -d --name hgg-ci -p 18790:8790 "$IMAGE:latest"
          for i in $(seq 1 20); do curl -sf localhost:18790/health >/dev/null 2>&1 && break; sleep 1; done
          MUD_URL=ws://localhost:18790/ws timeout 90 node "$SMOKE" > smoke.txt 2>&1 || true
          docker rm -f hgg-ci >/dev/null 2>&1 || true
          pass=$(grep -c "^ok" smoke.txt || true)
          fail=$(grep -c "^FAIL" smoke.txt || true)
          echo "smoke conformance: ${pass} passed / ${fail} failed (informational; the port is in progress)"
        '''
      }
      post {
        always {
          archiveArtifacts artifacts: 'smoke.txt', allowEmptyArchive: true
          sh 'docker rm -f hgg-ci >/dev/null 2>&1 || true'
        }
      }
    }

    stage('Deploy') {
      when { branch 'main' }
      steps {
        sh '''
          set -e
          docker rm -f hollow-grid-go >/dev/null 2>&1 || true
          docker run -d --restart unless-stopped --name hollow-grid-go \
            -p 8790:8790 -v hollow-grid-go-data:/data "$IMAGE:latest"
          echo "deployed hollow-grid-go on :8790 (named volume hollow-grid-go-data)"
        '''
      }
    }
  }

  post {
    always { sh 'docker rm -f hgg-ci >/dev/null 2>&1 || true' }
    success { echo 'pipeline green' }
  }
}
