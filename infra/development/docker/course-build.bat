set CGO_ENABLED=1
set GOOS=linux
set GOARCH=amd64
go build -o build/course-service ./services/course-service/cmd/main.go