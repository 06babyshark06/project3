# Load the restart_process extension
load('ext://restart_process', 'docker_build_with_restart')

### K8s Config ###

# Uncomment to use secrets
# k8s_yaml('./infra/development/k8s/secrets.yaml')

k8s_yaml('./infra/development/k8s/app-config.yaml')

### End of K8s Config ###
### API Gateway ###

gateway_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-gateway ./services/api-gateway'
if os.name == 'nt':
  gateway_compile_cmd = './infra/development/docker/api-gateway-build.bat'

local_resource(
  'api-gateway-compile',
  gateway_compile_cmd,
  deps=['./services/api-gateway', './shared'], labels="compiles")


docker_build_with_restart(
  'JQKStudy/api-gateway',
  '.',
  entrypoint=['/app/build/api-gateway'],
  dockerfile='./infra/development/docker/api-gateway.Dockerfile',
  only=[
    './build/api-gateway',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/api-gateway-deployment.yaml')
k8s_resource('api-gateway', port_forwards=8081,
             resource_deps=['api-gateway-compile'], labels="services")
### End of API Gateway ###
### User Service ###


user_compile_cmd = 'CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/user-service ./services/user-service/cmd/main.go'
if os.name == 'nt':
 user_compile_cmd = './infra/development/docker/user-build.bat'

local_resource(
  'user-service-compile',
  user_compile_cmd,
  deps=['./services/user-service', './shared'], labels="compiles")

docker_build_with_restart(
  'JQKStudy/user-service',
  '.',
  entrypoint=['/app/build/user-service'],
  dockerfile='./infra/development/docker/user-service.Dockerfile',
  only=[
    './build/user-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/user-service-deployment.yaml')
k8s_resource('user-service', resource_deps=['user-service-compile', 'postgres'], labels="services")

### End of User Service ###
### Exam Service ###


exam_compile_cmd = 'CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/exam-service ./services/exam-service/cmd/main.go'
if os.name == 'nt':
 exam_compile_cmd = './infra/development/docker/exam-build.bat'

local_resource(
  'exam-service-compile',
  exam_compile_cmd,
  deps=['./services/exam-service', './shared'], labels="compiles")

docker_build_with_restart(
  'JQKStudy/exam-service',
  '.',
  entrypoint=['/app/build/exam-service'],
  dockerfile='./infra/development/docker/exam-service.Dockerfile',
  only=[
    './build/exam-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/exam-service-deployment.yaml')
k8s_resource('exam-service', resource_deps=['exam-service-compile', 'postgres'], labels="services")

### End of Exam Service ###
### Course Service ###


course_compile_cmd = 'CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/course-service ./services/course-service/cmd/main.go'
if os.name == 'nt':
 course_compile_cmd = './infra/development/docker/course-build.bat'

local_resource(
  'course-service-compile',
  course_compile_cmd,
  deps=['./services/course-service', './shared'], labels="compiles")

docker_build_with_restart(
  'JQKStudy/course-service',
  '.',
  entrypoint=['/app/build/course-service'],
  dockerfile='./infra/development/docker/course-service.Dockerfile',
  only=[
    './build/course-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/course-service-deployment.yaml')
k8s_resource('course-service', resource_deps=['course-service-compile', 'postgres'], labels="services")

### End of Course Service ###
### Notification Service ###


notification_compile_cmd = 'CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/notification-service ./services/notification-service/cmd/main.go'
if os.name == 'nt':
 notification_compile_cmd = './infra/development/docker/notification-build.bat'

local_resource(
  'notification-service-compile',
  notification_compile_cmd,
  deps=['./services/notification-service', './shared'], labels="compiles")

docker_build_with_restart(
  'JQKStudy/notification-service',
  '.',
  entrypoint=['/app/build/notification-service'],
  dockerfile='./infra/development/docker/notification-service.Dockerfile',
  only=[
    './build/notification-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/notification-service-deployment.yaml')
k8s_resource('notification-service', resource_deps=['notification-service-compile', 'postgres'], labels="services")

### End of Course Service ###
### Web Frontend ###

# docker_build(
#   'ride-sharing/web',
#   '.',
#   dockerfile='./infra/development/docker/web.Dockerfile',
# )

# k8s_yaml('./infra/development/k8s/web-deployment.yaml')
# k8s_resource('web', port_forwards=3000, labels="frontend")

### End of Web Frontend ###

### Database ###
docker_build(
  'JQKStudy/postgres',
  '.',
  dockerfile='./infra/development/docker/postgres.Dockerfile',
)

k8s_yaml('./infra/development/k8s/postgres-deployment.yaml')
k8s_resource('postgres', port_forwards=5432, labels="postgres")

### Redis ###
docker_build(
  'JQKStudy/redis',
  '.',
  dockerfile='./infra/development/docker/redis.Dockerfile',
)

# k8s_yaml('./infra/development/k8s/redis-pvc.yaml')
k8s_yaml('./infra/development/k8s/redis-deployment.yaml')

k8s_resource('redis', port_forwards=6379, labels="redis")
### End of Redis ###