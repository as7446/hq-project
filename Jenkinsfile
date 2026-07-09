/* 模版化配置，定义变量
   账号密钥均存放在凭证保证安全
*/
def CFG = [
  gitUrl              : 'git@github.com:as7446/hq-project.git',
  gitBranch           : 'main',
  gitCredentials      : 'github-ssh-key',

  appName             : 'hq-project-demo',
  dockerfile          : 'Dockerfile',
  imageRegistry       : 'registry.cn-shanghai.aliyuncs.com',
  imageNamespace      : 'sh-cloud',
  registryCredentials : 'aliyun-acr-credential',

  deployHost          : '47.76.29.16',
  deployUser          : 'deploy',
  deploySshCredential : 'prod-deploy-ssh-key',
  remoteDeployDir     : '/data/hp/hq-project-demo',
  composeFile         : 'deploy/app/docker-compose.yml',

  appEnv              : 'prod',
  httpPort            : '8080'
]

pipeline {
  agent any

  options {
    timestamps()
    buildDiscarder(logRotator(numToKeepStr: '20'))
    disableConcurrentBuilds()
    timeout(time: 30, unit: 'MINUTES')
  }

  environment {
  IMAGE_REPO = "${CFG.imageRegistry}/${CFG.imageNamespace}/${CFG.appName}"
  LATEST_IMAGE = "${CFG.imageRegistry}/${CFG.imageNamespace}/${CFG.appName}:latest"
  DOCKER_BUILDKIT = '0'

  APP_ENV_VALUE = "${CFG.appEnv}"
  APP_PORT = "${CFG.httpPort}"
  IMAGE_REGISTRY = "${CFG.imageRegistry}"
  DEPLOY_HOST = "${CFG.deployHost}"
  DEPLOY_USER = "${CFG.deployUser}"
  REMOTE_DEPLOY_DIR = "${CFG.remoteDeployDir}"
  COMPOSE_FILE = "${CFG.composeFile}"
  DOCKERFILE_PATH = "${CFG.dockerfile}"
}

  stages {
  // 拉去代码
    stage('Checkout') {
      steps {
        checkout([
          $class: 'GitSCM',
          branches: [[name: "*/${CFG.gitBranch}"]],
          userRemoteConfigs: [[
            url: CFG.gitUrl,
            credentialsId: CFG.gitCredentials
          ]]
        ])

        script {
          env.GIT_COMMIT_FULL = sh(script: 'git rev-parse HEAD', returnStdout: true).trim()
          env.GIT_COMMIT_SHORT = sh(script: 'git rev-parse --short=12 HEAD', returnStdout: true).trim()
          env.IMAGE_TAG = "${env.BUILD_NUMBER}-${env.GIT_COMMIT_SHORT}"
          env.FULL_IMAGE = "${env.IMAGE_REPO}:${env.IMAGE_TAG}"

          echo "Git Commit: ${env.GIT_COMMIT_FULL}"
          echo "Image: ${env.FULL_IMAGE}"
        }
      }
    }
    // 分发配置
    stage('Render Config') {
      steps {
        sh '''
          set -eu

          mkdir -p .jenkins/rendered

          cat > .jenkins/rendered/app.env <<EOF
SERVICE_NAME=${JOB_NAME}
APP_ENV=${APP_ENV_VALUE}
PORT=${APP_PORT}
JOB_NAME=${JOB_NAME}
BUILD_NUMBER=${BUILD_NUMBER}
BUILD_URL=${BUILD_URL}
GIT_COMMIT=${GIT_COMMIT_FULL}
GIT_COMMIT_SHORT=${GIT_COMMIT_SHORT}
IMAGE=${FULL_IMAGE}
EOF

          echo "Rendered .env:"
          cat .jenkins/rendered/app.env
        '''

        archiveArtifacts artifacts: '.jenkins/rendered/app.env', fingerprint: true
      }
    }

    // 单元测试
    stage('Unit Test') {
      steps {
        sh '''
          set -eu

          docker run --rm \
            --volumes-from "$(hostname)" \
            -w "${WORKSPACE}" \
            -e GOPROXY=https://goproxy.cn,direct \
            -e PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
            golang:1.23 \
            sh -lc '
              set -eu

              echo "Current dir:"
              pwd

              echo "Files:"
              ls -la

              echo "PATH:"
              echo "$PATH"

              echo "Check go binary:"
              ls -l /usr/local/go/bin/go
              /usr/local/go/bin/go version

              echo "Go module:"
              /usr/local/go/bin/go env GOMOD

              if [ ! -f go.mod ]; then
                echo "ERROR: go.mod not found"
                exit 1
              fi

              /usr/local/go/bin/go test ./... -count=1 -coverprofile=coverage.out
            '
        '''

        archiveArtifacts artifacts: 'coverage.out', allowEmptyArchive: true
      }
    }

    // 构建镜像
    stage('Build Image') {
      steps {
        sh '''
          set -eu

          DOCKER_BUILDKIT=0 docker build \
            --pull \
            --build-arg VERSION="${JOB_NAME}#${BUILD_NUMBER}@${GIT_COMMIT_SHORT}" \
            --build-arg GIT_COMMIT="${GIT_COMMIT_FULL}" \
            --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            -f "${DOCKERFILE_PATH}" \
            -t "${FULL_IMAGE}" \
            -t "${LATEST_IMAGE}" .
        '''
      }
    }

    // 上传镜像制品
    stage('Push Image') {
      steps {
        withCredentials([
          usernamePassword(
            credentialsId: CFG.registryCredentials,
            usernameVariable: 'REGISTRY_USER',
            passwordVariable: 'REGISTRY_PASS'
          )
        ]) {
          sh '''
            set -eu
            set +x

            echo "${REGISTRY_PASS}" | docker login "${IMAGE_REGISTRY}" \
              -u "${REGISTRY_USER}" \
              --password-stdin

            set -x

            docker push "${FULL_IMAGE}"
            docker push "${LATEST_IMAGE}"
          '''
        }
      }
    }

    // 部署服务，服务管理docker代替supervisor了，因为是演示所以直接在piplie部署了，实际环境服务状态应该发布到git（gitops），由控制器完成升级
    stage('Deploy Compose') {

      steps {

        withCredentials([
          sshUserPrivateKey(
            credentialsId: CFG.deploySshCredential,
            keyFileVariable: 'SSH_KEY',
            usernameVariable: 'SSH_USER'
          )
        ]) {

          sh '''
            set -eu

            chmod 600 "$SSH_KEY"

            echo "Prepare remote directory..."

            ssh \
              -i "$SSH_KEY" \
              -o StrictHostKeyChecking=accept-new \
              ${SSH_USER}@${DEPLOY_HOST} \
              "mkdir -p ${REMOTE_DEPLOY_DIR}"


            echo "Upload docker compose..."

            scp \
              -i "$SSH_KEY" \
              ${COMPOSE_FILE} \
              ${SSH_USER}@${DEPLOY_HOST}:${REMOTE_DEPLOY_DIR}/docker-compose.yml


            echo "Upload env..."

            scp \
              -i "$SSH_KEY" \
              .jenkins/rendered/app.env \
              ${SSH_USER}@${DEPLOY_HOST}:${REMOTE_DEPLOY_DIR}/.env


            echo "Deploy..."

            ssh \
              -i "$SSH_KEY" \
              ${SSH_USER}@${DEPLOY_HOST} \
              "
              cd ${REMOTE_DEPLOY_DIR}

              docker compose pull

              docker compose up -d --remove-orphans
              "
          '''
        }
      }
    }
  }

// 这里面可以根据构建状态发送不同的通知到群组，演示不做配置了
  post {
    success {
      echo 'Pipeline SUCCESS'
    }

    failure {
      echo 'Pipeline FAILURE'
    }

    always {
      sh '''
        docker logout "${IMAGE_REGISTRY}" || true
        docker image prune -f --filter "until=72h" || true
      '''
    }
  }
}