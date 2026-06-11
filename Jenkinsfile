// CI/CD for hollow-grid-go (a Hollow Grid world server in Go).
//
// On every push: gofmt + go vet, unit tests, build the Docker image, and run the
// upstream smoke conformance suite against the freshly-built container
// (informational while the port is in progress, so a partial pass never reds the
// build). Deploy is a separate, deliberate step (push + pin image on the target
// host) -- the fleet build hosts are ephemeral agents, not the run target.
//
// Agent: fleet `build` label (fugazi/jello/damaged). Go and Docker CLI are in
// the agent image; the host Docker daemon is reached via the bind-mounted socket.
// Smoke suite: cloned from skyphusion-labs/the-hollow-grid at HEAD alongside the
// build checkout (no hardcoded host paths).
pipeline {
  agent { label 'build' }

  options {
    timestamps()
    timeout(time: 15, unit: 'MINUTES')
    disableConcurrentBuilds()
  }

  environment {
    IMAGE = 'ghcr.io/skyphusion/hollow-grid-go'
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
          SMOKE_DIR=$(mktemp -d)
          git clone --depth 1 https://github.com/skyphusion-labs/the-hollow-grid.git "$SMOKE_DIR" 2>/dev/null || true
          SMOKE="$SMOKE_DIR/smoke.mjs"
          if [ ! -f "$SMOKE" ]; then echo "SKIP: smoke suite not found"; exit 0; fi
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

    stage('Push (main)') {
      // Push the built image to GHCR on main so the run target can pull it.
      // Actual (re)deploy is a separate step: pull + restart on the target host.
      when { branch 'main' }
      steps {
        withCredentials([usernamePassword(
          credentialsId: 'ghcr-skyphusion',
          usernameVariable: 'GHCR_USER',
          passwordVariable: 'GHCR_TOKEN',
        )]) {
          sh '''
            echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USER" --password-stdin
            docker push "$IMAGE:$GIT_COMMIT"
            docker push "$IMAGE:latest"
          '''
        }
      }
    }
  }

  post {
    always {
      sh 'docker rm -f hgg-ci >/dev/null 2>&1 || true'
      sh 'docker logout ghcr.io || true'
    }
    success { echo 'pipeline green' }
  }
}
