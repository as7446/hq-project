
/*
    定义环境变量
*/
def CFG = [
  gitUrl              : 'https://github.com/as7446/hq-project.git',
  gitBranch           : 'main',
  appName             : 'hq-project-demo',
  dockerfile          : 'Dockerfile',
  imageRegistry       : 'registry.cn-hangzhou.aliyuncs.com',
  imageNamespace      : 'sh-cloud',
  registryCredentials : 'docker-registry-credentials-id',
  deployHost          : 'prod-docker-01.gowellgo.top',
  deployUser          : 'deploy',
  deploySshCredential : 'prod-deploy-ssh-key-id',
  remoteDeployDir     : '/opt/hq-project-demo',
  composeFile         : 'deploy/app/docker-compose.yml',
  appEnv              : 'prod',
  httpPort            : '8080',
  wecomWebhook        : ''
]

/*
    执行过程
*/
pipeline {
  agent any

  options {
    timestamps()
    ansiColor('xterm')
    buildDiscarder(logRotator(numToKeepStr: '20'))
    disableConcurrentBuilds()
    timeout(time: 30, unit: 'MINUTES')
  }

  environment {
    IMAGE_REPO = "${CFG.imageRegistry}/${CFG.imageNamespace}/${CFG.appName}"
    IMAGE_TAG = "${env.BUILD_NUMBER}-${env.GIT_COMMIT ?: 'manual'}"
    FULL_IMAGE = "${env.IMAGE_REPO}:${env.IMAGE_TAG}"
    LATEST_IMAGE = "${env.IMAGE_REPO}:latest"
    DOCKER_BUILDKIT = '1'
  }


  stages {
   // 拉取代码
    stage('Checkout') {
      steps {
        checkout([$class: 'GitSCM',
          branches: [[name: "*/${CFG.gitBranch}"]],
          userRemoteConfigs: [[url: CFG.gitUrl]]
        ])
        script {
          env.GIT_COMMIT_SHORT = sh(script: 'git rev-parse --short=12 HEAD', returnStdout: true).trim()
          env.IMAGE_TAG = "${env.BUILD_NUMBER}-${env.GIT_COMMIT_SHORT}"
          env.FULL_IMAGE = "${env.IMAGE_REPO}:${env.IMAGE_TAG}"
        }
      }
    }

    // 配置分发
    stage('Render Config') {
      steps {
        sh """
          set -eu
          mkdir -p .jenkins/rendered
          cat > .jenkins/rendered/app.env <<EOF
SERVICE_NAME=${JOB_NAME}
APP_ENV=${CFG.appEnv}
PORT=${CFG.httpPort}
JOB_NAME=${JOB_NAME}
BUILD_NUMBER=${BUILD_NUMBER}
BUILD_URL=${BUILD_URL}
GIT_COMMIT=${GIT_COMMIT}
IMAGE=${FULL_IMAGE}
EOF
        """
        archiveArtifacts artifacts: '.jenkins/rendered/app.env', fingerprint: true
      }
    }

    // 单元测试
    stage('Unit Test') {
      steps {
        sh '''
          set -eu
          go version
          go test ./... -count=1 -coverprofile=coverage.out
        '''
        archiveArtifacts artifacts: 'coverage.out', allowEmptyArchive: true
      }
    }

    // 构建镜像
    stage('Build Image') {
      steps {
        sh """
          set -eu
          docker build \
            --pull \
            --build-arg VERSION="${JOB_NAME}#${BUILD_NUMBER}@${GIT_COMMIT_SHORT}" \
            --build-arg GIT_COMMIT="${GIT_COMMIT}" \
            --build-arg BUILD_TIME="\$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            -f "${CFG.dockerfile}" \
            -t "${FULL_IMAGE}" \
            -t "${LATEST_IMAGE}" .
        """
      }
    }

    // 上传镜像制品
    stage('Push Image') {
      steps {
        withCredentials([usernamePassword(credentialsId: CFG.registryCredentials, usernameVariable: 'REGISTRY_USER', passwordVariable: 'REGISTRY_PASS')]) {
          sh """
            set +x
            echo "\${REGISTRY_PASS}" | docker login "${CFG.imageRegistry}" -u "\${REGISTRY_USER}" --password-stdin
            set -x
            docker push "${FULL_IMAGE}"
            docker push "${LATEST_IMAGE}"
            docker logout "${CFG.imageRegistry}"
          """
        }
      }
    }

    /*
        这只是一个简单的流程演示环境所以使用docker compose，生成环境这部分应该是更新至git，由控制器触发滚动升级
    */
    stage('Deploy Compose') {
      steps {
        sshagent(credentials: [CFG.deploySshCredential]) {
          sh """
            set -eu
            ssh -o StrictHostKeyChecking=accept-new ${CFG.deployUser}@${CFG.deployHost} "mkdir -p ${CFG.remoteDeployDir}"
            scp "${CFG.composeFile}" ${CFG.deployUser}@${CFG.deployHost}:${CFG.remoteDeployDir}/docker-compose.yml
            scp .jenkins/rendered/app.env ${CFG.deployUser}@${CFG.deployHost}:${CFG.remoteDeployDir}/.env
            ssh ${CFG.deployUser}@${CFG.deployHost} "cd ${CFG.remoteDeployDir} && docker compose pull && docker compose up -d --remove-orphans"
          """
        }
      }
    }
  }
  // 不同状态执行不同告警通知，演示先注释掉了
  post {
    success {
      script {
        //notifyWeCom(CFG.wecomWebhook, "SUCCESS", env.FULL_IMAGE)
      }
    }
    failure {
      script {
        //notifyWeCom(CFG.wecomWebhook, "FAILURE", env.FULL_IMAGE)
      }
    }
    // 清理释放空间
    always {
      sh 'docker image prune -f --filter "until=72h" || true'
    }
  }
}
// 根据构建状态发送通知到企业微信，演示就不接入了
def notifyWeCom(String webhook, String status, String image) {
  if (!webhook?.trim()) {
    echo "WeCom webhook is empty, skip notification. status=${status}, image=${image}"
    return
  }
  def text = "Project: ${env.JOB_NAME}\\nBuild: #${env.BUILD_NUMBER}\\nStatus: ${status}\\nImage: ${image}\\nURL: ${env.BUILD_URL}"
  sh """
    curl -sS -X POST '${webhook}' \
      -H 'Content-Type: application/json' \
      -d '{"msgtype":"text","text":{"content":"${text}"}}'
  """
}
